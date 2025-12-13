package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// commandRunner abstracts exec.Cmd for testability
type commandRunner interface {
	Start() error
	Wait() error
	StdoutPipe() (io.ReadCloser, error)
	StdinPipe() (io.WriteCloser, error)
	SetStderr(io.Writer)
	SetDir(string)
	Process() processHandle
}

// processHandle abstracts os.Process for testability
type processHandle interface {
	Pid() int
	Kill() error
	Signal(os.Signal) error
}

// realCmd implements commandRunner using exec.Cmd
type realCmd struct {
	cmd *exec.Cmd
}

func (r *realCmd) Start() error {
	if r.cmd == nil {
		return errors.New("command is nil")
	}
	return r.cmd.Start()
}

func (r *realCmd) Wait() error {
	if r.cmd == nil {
		return errors.New("command is nil")
	}
	return r.cmd.Wait()
}

func (r *realCmd) StdoutPipe() (io.ReadCloser, error) {
	if r.cmd == nil {
		return nil, errors.New("command is nil")
	}
	return r.cmd.StdoutPipe()
}

func (r *realCmd) StdinPipe() (io.WriteCloser, error) {
	if r.cmd == nil {
		return nil, errors.New("command is nil")
	}
	return r.cmd.StdinPipe()
}

func (r *realCmd) SetStderr(w io.Writer) {
	if r.cmd != nil {
		r.cmd.Stderr = w
	}
}

func (r *realCmd) SetDir(dir string) {
	if r.cmd != nil {
		r.cmd.Dir = dir
	}
}

func (r *realCmd) Process() processHandle {
	if r == nil || r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	return &realProcess{proc: r.cmd.Process}
}

// realProcess implements processHandle using os.Process
type realProcess struct {
	proc *os.Process
}

func (p *realProcess) Pid() int {
	if p == nil || p.proc == nil {
		return 0
	}
	return p.proc.Pid
}

func (p *realProcess) Kill() error {
	if p == nil || p.proc == nil {
		return nil
	}
	return p.proc.Kill()
}

func (p *realProcess) Signal(sig os.Signal) error {
	if p == nil || p.proc == nil {
		return nil
	}
	return p.proc.Signal(sig)
}

// newCommandRunner creates a new commandRunner (test hook injection point)
var newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
	return &realCmd{cmd: commandContext(ctx, name, args...)}
}

type parseResult struct {
	message  string
	threadID string
}

var runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
	if task.WorkDir == "" {
		task.WorkDir = defaultWorkdir
	}
	if task.Mode == "" {
		task.Mode = "new"
	}
	if task.UseStdin || shouldUseStdin(task.Task, false) {
		task.UseStdin = true
	}

	backendName := task.Backend
	if backendName == "" {
		backendName = defaultBackendName
	}

	backend, err := selectBackendFn(backendName)
	if err != nil {
		return TaskResult{TaskID: task.ID, ExitCode: 1, Error: err.Error()}
	}
	task.Backend = backend.Name()

	parentCtx := task.Context
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	return runCodexTaskWithContext(parentCtx, task, backend, nil, false, true, timeout)
}

func topologicalSort(tasks []TaskSpec) ([][]TaskSpec, error) {
	idToTask := make(map[string]TaskSpec, len(tasks))
	indegree := make(map[string]int, len(tasks))
	adj := make(map[string][]string, len(tasks))

	for _, task := range tasks {
		idToTask[task.ID] = task
		indegree[task.ID] = 0
	}

	for _, task := range tasks {
		for _, dep := range task.Dependencies {
			if _, ok := idToTask[dep]; !ok {
				return nil, fmt.Errorf("dependency %q not found for task %q", dep, task.ID)
			}
			indegree[task.ID]++
			adj[dep] = append(adj[dep], task.ID)
		}
	}

	queue := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if indegree[task.ID] == 0 {
			queue = append(queue, task.ID)
		}
	}

	layers := make([][]TaskSpec, 0)
	processed := 0

	for len(queue) > 0 {
		current := queue
		queue = nil
		layer := make([]TaskSpec, len(current))
		for i, id := range current {
			layer[i] = idToTask[id]
			processed++
		}
		layers = append(layers, layer)

		next := make([]string, 0)
		for _, id := range current {
			for _, neighbor := range adj[id] {
				indegree[neighbor]--
				if indegree[neighbor] == 0 {
					next = append(next, neighbor)
				}
			}
		}
		queue = append(queue, next...)
	}

	if processed != len(tasks) {
		cycleIDs := make([]string, 0)
		for id, deg := range indegree {
			if deg > 0 {
				cycleIDs = append(cycleIDs, id)
			}
		}
		sort.Strings(cycleIDs)
		return nil, fmt.Errorf("cycle detected involving tasks: %s", strings.Join(cycleIDs, ","))
	}

	return layers, nil
}

func executeConcurrent(layers [][]TaskSpec, timeout int) []TaskResult {
	maxWorkers := resolveMaxParallelWorkers()
	return executeConcurrentWithContext(context.Background(), layers, timeout, maxWorkers)
}

func executeConcurrentWithContext(parentCtx context.Context, layers [][]TaskSpec, timeout int, maxWorkers int) []TaskResult {
	totalTasks := 0
	for _, layer := range layers {
		totalTasks += len(layer)
	}

	results := make([]TaskResult, 0, totalTasks)
	failed := make(map[string]TaskResult, totalTasks)
	resultsCh := make(chan TaskResult, totalTasks)

	var startPrintMu sync.Mutex
	bannerPrinted := false

	printTaskStart := func(taskID string) {
		logger := activeLogger()
		if logger == nil {
			return
		}
		path := logger.Path()
		if path == "" {
			return
		}
		startPrintMu.Lock()
		if !bannerPrinted {
			fmt.Fprintln(os.Stderr, "=== Starting Parallel Execution ===")
			bannerPrinted = true
		}
		fmt.Fprintf(os.Stderr, "Task %s: Log: %s\n", taskID, path)
		startPrintMu.Unlock()
	}

	ctx := parentCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerLimit := maxWorkers
	if workerLimit < 0 {
		workerLimit = 0
	}

	var sem chan struct{}
	if workerLimit > 0 {
		sem = make(chan struct{}, workerLimit)
	}

	logConcurrencyPlanning(workerLimit, totalTasks)

	acquireSlot := func() bool {
		if sem == nil {
			return true
		}
		select {
		case sem <- struct{}{}:
			return true
		case <-ctx.Done():
			return false
		}
	}

	releaseSlot := func() {
		if sem == nil {
			return
		}
		select {
		case <-sem:
		default:
		}
	}

	var activeWorkers int64

	for _, layer := range layers {
		var wg sync.WaitGroup
		executed := 0

		for _, task := range layer {
			if skip, reason := shouldSkipTask(task, failed); skip {
				res := TaskResult{TaskID: task.ID, ExitCode: 1, Error: reason}
				results = append(results, res)
				failed[task.ID] = res
				continue
			}

			if ctx.Err() != nil {
				res := cancelledTaskResult(task.ID, ctx)
				results = append(results, res)
				failed[task.ID] = res
				continue
			}

			executed++
			wg.Add(1)
			go func(ts TaskSpec) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						resultsCh <- TaskResult{TaskID: ts.ID, ExitCode: 1, Error: fmt.Sprintf("panic: %v", r)}
					}
				}()

				if !acquireSlot() {
					resultsCh <- cancelledTaskResult(ts.ID, ctx)
					return
				}
				defer releaseSlot()

				current := atomic.AddInt64(&activeWorkers, 1)
				logConcurrencyState("start", ts.ID, int(current), workerLimit)
				defer func() {
					after := atomic.AddInt64(&activeWorkers, -1)
					logConcurrencyState("done", ts.ID, int(after), workerLimit)
				}()

				ts.Context = ctx
				printTaskStart(ts.ID)
				resultsCh <- runCodexTaskFn(ts, timeout)
			}(task)
		}

		wg.Wait()

		for i := 0; i < executed; i++ {
			res := <-resultsCh
			results = append(results, res)
			if res.ExitCode != 0 || res.Error != "" {
				failed[res.TaskID] = res
			}
		}
	}

	return results
}

func cancelledTaskResult(taskID string, ctx context.Context) TaskResult {
	exitCode := 130
	msg := "execution cancelled"
	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		exitCode = 124
		msg = "execution timeout"
	}
	return TaskResult{TaskID: taskID, ExitCode: exitCode, Error: msg}
}

func shouldSkipTask(task TaskSpec, failed map[string]TaskResult) (bool, string) {
	if len(task.Dependencies) == 0 {
		return false, ""
	}

	var blocked []string
	for _, dep := range task.Dependencies {
		if _, ok := failed[dep]; ok {
			blocked = append(blocked, dep)
		}
	}

	if len(blocked) == 0 {
		return false, ""
	}

	return true, fmt.Sprintf("skipped due to failed dependencies: %s", strings.Join(blocked, ","))
}

func generateFinalOutput(results []TaskResult) string {
	var sb strings.Builder

	success := 0
	failed := 0
	for _, res := range results {
		if res.ExitCode == 0 && res.Error == "" {
			success++
		} else {
			failed++
		}
	}

	sb.WriteString(fmt.Sprintf("=== Parallel Execution Summary ===\n"))
	sb.WriteString(fmt.Sprintf("Total: %d | Success: %d | Failed: %d\n\n", len(results), success, failed))

	for _, res := range results {
		sb.WriteString(fmt.Sprintf("--- Task: %s ---\n", res.TaskID))
		if res.Error != "" {
			sb.WriteString(fmt.Sprintf("Status: FAILED (exit code %d)\nError: %s\n", res.ExitCode, res.Error))
		} else if res.ExitCode != 0 {
			sb.WriteString(fmt.Sprintf("Status: FAILED (exit code %d)\n", res.ExitCode))
		} else {
			sb.WriteString("Status: SUCCESS\n")
		}
		if res.SessionID != "" {
			sb.WriteString(fmt.Sprintf("Session: %s\n", res.SessionID))
		}
		if res.LogPath != "" {
			sb.WriteString(fmt.Sprintf("Log: %s\n", res.LogPath))
		}
		if res.Message != "" {
			sb.WriteString(fmt.Sprintf("\n%s\n", res.Message))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func buildCodexArgs(cfg *Config, targetArg string) []string {
	if cfg.Mode == "resume" {
		return []string{
			"e",
			"--skip-git-repo-check",
			"--json",
			"resume",
			cfg.SessionID,
			targetArg,
		}
	}
	return []string{
		"e",
		"--skip-git-repo-check",
		"-C", cfg.WorkDir,
		"--json",
		targetArg,
	}
}

func runCodexTask(taskSpec TaskSpec, silent bool, timeoutSec int) TaskResult {
	return runCodexTaskWithContext(context.Background(), taskSpec, nil, nil, false, silent, timeoutSec)
}

func runCodexProcess(parentCtx context.Context, codexArgs []string, taskText string, useStdin bool, timeoutSec int) (message, threadID string, exitCode int) {
	res := runCodexTaskWithContext(parentCtx, TaskSpec{Task: taskText, WorkDir: defaultWorkdir, Mode: "new", UseStdin: useStdin}, nil, codexArgs, true, false, timeoutSec)
	return res.Message, res.SessionID, res.ExitCode
}

func runCodexTaskWithContext(parentCtx context.Context, taskSpec TaskSpec, backend Backend, customArgs []string, useCustomArgs bool, silent bool, timeoutSec int) TaskResult {
	result := TaskResult{TaskID: taskSpec.ID}
	setLogPath := func() {
		if result.LogPath != "" {
			return
		}
		if logger := activeLogger(); logger != nil {
			result.LogPath = logger.Path()
		}
	}

	cfg := &Config{
		Mode:      taskSpec.Mode,
		Task:      taskSpec.Task,
		SessionID: taskSpec.SessionID,
		WorkDir:   taskSpec.WorkDir,
		Backend:   defaultBackendName,
	}

	commandName := codexCommand
	argsBuilder := buildCodexArgsFn
	if backend != nil {
		commandName = backend.Command()
		argsBuilder = backend.BuildArgs
		cfg.Backend = backend.Name()
	} else if taskSpec.Backend != "" {
		cfg.Backend = taskSpec.Backend
	} else if commandName != "" {
		cfg.Backend = commandName
	}

	if cfg.Mode == "" {
		cfg.Mode = "new"
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = defaultWorkdir
	}

	useStdin := taskSpec.UseStdin
	targetArg := taskSpec.Task
	if useStdin {
		targetArg = "-"
	}

	var codexArgs []string
	if useCustomArgs {
		codexArgs = customArgs
	} else {
		codexArgs = argsBuilder(cfg, targetArg)
	}

	prefixMsg := func(msg string) string {
		if taskSpec.ID == "" {
			return msg
		}
		return fmt.Sprintf("[Task: %s] %s", taskSpec.ID, msg)
	}

	var logInfoFn func(string)
	var logWarnFn func(string)
	var logErrorFn func(string)

	if silent {
		// Silent mode: only persist to file when available; avoid stderr noise.
		logInfoFn = func(msg string) {
			if logger := activeLogger(); logger != nil {
				logger.Info(prefixMsg(msg))
			}
		}
		logWarnFn = func(msg string) {
			if logger := activeLogger(); logger != nil {
				logger.Warn(prefixMsg(msg))
			}
		}
		logErrorFn = func(msg string) {
			if logger := activeLogger(); logger != nil {
				logger.Error(prefixMsg(msg))
			}
		}
	} else {
		logInfoFn = func(msg string) { logInfo(prefixMsg(msg)) }
		logWarnFn = func(msg string) { logWarn(prefixMsg(msg)) }
		logErrorFn = func(msg string) { logError(prefixMsg(msg)) }
	}

	stderrBuf := &tailBuffer{limit: stderrCaptureLimit}

	var stdoutLogger *logWriter
	var stderrLogger *logWriter

	var tempLogger *Logger
	if silent && activeLogger() == nil {
		if l, err := NewLogger(); err == nil {
			setLogger(l)
			tempLogger = l
		}
	}
	defer func() {
		if tempLogger != nil {
			_ = closeLogger()
		}
	}()
	defer setLogPath()
	if logger := activeLogger(); logger != nil {
		result.LogPath = logger.Path()
	}

	if !silent {
		stdoutLogger = newLogWriter("CODEX_STDOUT: ", codexLogLineLimit)
		stderrLogger = newLogWriter("CODEX_STDERR: ", codexLogLineLimit)
	}

	ctx := parentCtx
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	attachStderr := func(msg string) string {
		return fmt.Sprintf("%s; stderr: %s", msg, stderrBuf.String())
	}

	cmd := newCommandRunner(ctx, commandName, codexArgs...)

	// For backends that don't support -C flag (claude, gemini), set working directory via cmd.Dir
	// Codex passes workdir via -C flag, so we skip setting Dir for it to avoid conflicts
	if cfg.Mode != "resume" && commandName != "codex" && cfg.WorkDir != "" {
		cmd.SetDir(cfg.WorkDir)
	}

	stderrWriters := []io.Writer{stderrBuf}
	if stderrLogger != nil {
		stderrWriters = append(stderrWriters, stderrLogger)
	}
	if !silent {
		stderrWriters = append([]io.Writer{os.Stderr}, stderrWriters...)
	}
	if len(stderrWriters) == 1 {
		cmd.SetStderr(stderrWriters[0])
	} else {
		cmd.SetStderr(io.MultiWriter(stderrWriters...))
	}

	var stdinPipe io.WriteCloser
	var err error
	if useStdin {
		stdinPipe, err = cmd.StdinPipe()
		if err != nil {
			logErrorFn("Failed to create stdin pipe: " + err.Error())
			result.ExitCode = 1
			result.Error = attachStderr("failed to create stdin pipe: " + err.Error())
			return result
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logErrorFn("Failed to create stdout pipe: " + err.Error())
		result.ExitCode = 1
		result.Error = attachStderr("failed to create stdout pipe: " + err.Error())
		return result
	}

	stdoutReader := io.Reader(stdout)
	if stdoutLogger != nil {
		stdoutReader = io.TeeReader(stdout, stdoutLogger)
	}

	// Start parse goroutine BEFORE starting the command to avoid race condition
	// where fast-completing commands close stdout before parser starts reading
	messageSeen := make(chan struct{}, 1)
	parseCh := make(chan parseResult, 1)
	go func() {
		msg, tid := parseJSONStreamInternal(stdoutReader, logWarnFn, logInfoFn, func() {
			select {
			case messageSeen <- struct{}{}:
			default:
			}
		})
		parseCh <- parseResult{message: msg, threadID: tid}
	}()

	logInfoFn(fmt.Sprintf("Starting %s with args: %s %s...", commandName, commandName, strings.Join(codexArgs[:min(5, len(codexArgs))], " ")))

	if err := cmd.Start(); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			msg := fmt.Sprintf("%s command not found in PATH", commandName)
			logErrorFn(msg)
			result.ExitCode = 127
			result.Error = attachStderr(msg)
			return result
		}
		logErrorFn("Failed to start " + commandName + ": " + err.Error())
		result.ExitCode = 1
		result.Error = attachStderr("failed to start " + commandName + ": " + err.Error())
		return result
	}

	logInfoFn(fmt.Sprintf("Starting %s with PID: %d", commandName, cmd.Process().Pid()))
	if logger := activeLogger(); logger != nil {
		logInfoFn(fmt.Sprintf("Log capturing to: %s", logger.Path()))
	}

	if useStdin && stdinPipe != nil {
		logInfoFn(fmt.Sprintf("Writing %d chars to stdin...", len(taskSpec.Task)))
		go func(data string) {
			defer stdinPipe.Close()
			_, _ = io.WriteString(stdinPipe, data)
		}(taskSpec.Task)
		logInfoFn("Stdin closed")
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	var waitErr error
	var forceKillTimer *forceKillTimer
	var ctxCancelled bool

	select {
	case waitErr = <-waitCh:
	case <-ctx.Done():
		ctxCancelled = true
		logErrorFn(cancelReason(commandName, ctx))
		forceKillTimer = terminateCommandFn(cmd)
		waitErr = <-waitCh
	}

	if forceKillTimer != nil {
		forceKillTimer.Stop()
	}

	var parsed parseResult
	if ctxCancelled {
		closeWithReason(stdout, stdoutCloseReasonCtx)
		parsed = <-parseCh
	} else {
		drainTimer := time.NewTimer(stdoutDrainTimeout)
		defer drainTimer.Stop()

		select {
		case parsed = <-parseCh:
			closeWithReason(stdout, stdoutCloseReasonWait)
		case <-messageSeen:
			closeWithReason(stdout, stdoutCloseReasonWait)
			parsed = <-parseCh
		case <-drainTimer.C:
			closeWithReason(stdout, stdoutCloseReasonDrain)
			parsed = <-parseCh
		}
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			result.ExitCode = 124
			result.Error = attachStderr(fmt.Sprintf("%s execution timeout", commandName))
			return result
		}
		result.ExitCode = 130
		result.Error = attachStderr("execution cancelled")
		return result
	}

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			logErrorFn(fmt.Sprintf("%s exited with status %d", commandName, code))
			result.ExitCode = code
			result.Error = attachStderr(fmt.Sprintf("%s exited with status %d", commandName, code))
			return result
		}
		logErrorFn(commandName + " error: " + waitErr.Error())
		result.ExitCode = 1
		result.Error = attachStderr(commandName + " error: " + waitErr.Error())
		return result
	}

	message := parsed.message
	threadID := parsed.threadID
	if message == "" {
		logErrorFn(fmt.Sprintf("%s completed without agent_message output", commandName))
		result.ExitCode = 1
		result.Error = attachStderr(fmt.Sprintf("%s completed without agent_message output", commandName))
		return result
	}

	if stdoutLogger != nil {
		stdoutLogger.Flush()
	}
	if stderrLogger != nil {
		stderrLogger.Flush()
	}

	result.ExitCode = 0
	result.Message = message
	result.SessionID = threadID
	if logger := activeLogger(); logger != nil {
		result.LogPath = logger.Path()
	}

	return result
}

func forwardSignals(ctx context.Context, cmd commandRunner, logErrorFn func(string)) {
	notify := signalNotifyFn
	stop := signalStopFn
	if notify == nil {
		notify = signal.Notify
	}
	if stop == nil {
		stop = signal.Stop
	}

	sigCh := make(chan os.Signal, 1)
	notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer stop(sigCh)
		select {
		case sig := <-sigCh:
			logErrorFn(fmt.Sprintf("Received signal: %v", sig))
			if proc := cmd.Process(); proc != nil {
				_ = proc.Signal(syscall.SIGTERM)
				time.AfterFunc(time.Duration(forceKillDelay.Load())*time.Second, func() {
					if p := cmd.Process(); p != nil {
						_ = p.Kill()
					}
				})
			}
		case <-ctx.Done():
		}
	}()
}

func cancelReason(commandName string, ctx context.Context) string {
	if ctx == nil {
		return "Context cancelled"
	}

	if commandName == "" {
		commandName = codexCommand
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Sprintf("%s execution timeout", commandName)
	}

	return fmt.Sprintf("Execution cancelled, terminating %s process", commandName)
}

type stdoutReasonCloser interface {
	CloseWithReason(string) error
}

func closeWithReason(rc io.ReadCloser, reason string) {
	if rc == nil {
		return
	}
	if c, ok := rc.(stdoutReasonCloser); ok {
		_ = c.CloseWithReason(reason)
		return
	}
	_ = rc.Close()
}

type forceKillTimer struct {
	timer   *time.Timer
	done    chan struct{}
	stopped atomic.Bool
	drained atomic.Bool
}

func (t *forceKillTimer) Stop() {
	if t == nil || t.timer == nil {
		return
	}
	if !t.timer.Stop() {
		<-t.done
		t.drained.Store(true)
	}
	t.stopped.Store(true)
}

func terminateCommand(cmd commandRunner) *forceKillTimer {
	if cmd == nil {
		return nil
	}
	proc := cmd.Process()
	if proc == nil {
		return nil
	}

	_ = proc.Signal(syscall.SIGTERM)

	done := make(chan struct{}, 1)
	timer := time.AfterFunc(time.Duration(forceKillDelay.Load())*time.Second, func() {
		if p := cmd.Process(); p != nil {
			_ = p.Kill()
		}
		close(done)
	})

	return &forceKillTimer{timer: timer, done: done}
}

func terminateProcess(cmd commandRunner) *time.Timer {
	if cmd == nil {
		return nil
	}
	proc := cmd.Process()
	if proc == nil {
		return nil
	}

	_ = proc.Signal(syscall.SIGTERM)

	return time.AfterFunc(time.Duration(forceKillDelay.Load())*time.Second, func() {
		if p := cmd.Process(); p != nil {
			_ = p.Kill()
		}
	})
}
