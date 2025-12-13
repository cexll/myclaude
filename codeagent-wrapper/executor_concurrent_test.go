package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type execFakeProcess struct {
	pid     int
	signals []os.Signal
	killed  atomic.Int32
	mu      sync.Mutex
}

func (p *execFakeProcess) Pid() int { return p.pid }
func (p *execFakeProcess) Kill() error {
	p.killed.Add(1)
	return nil
}
func (p *execFakeProcess) Signal(sig os.Signal) error {
	p.mu.Lock()
	p.signals = append(p.signals, sig)
	p.mu.Unlock()
	return nil
}

type writeCloserStub struct {
	bytes.Buffer
	closed atomic.Bool
}

func (w *writeCloserStub) Close() error {
	w.closed.Store(true)
	return nil
}

type reasonReadCloser struct {
	r       io.Reader
	closed  []string
	mu      sync.Mutex
	closedC chan struct{}
}

func newReasonReadCloser(data string) *reasonReadCloser {
	return &reasonReadCloser{r: strings.NewReader(data), closedC: make(chan struct{}, 1)}
}

func (rc *reasonReadCloser) Read(p []byte) (int, error) { return rc.r.Read(p) }
func (rc *reasonReadCloser) Close() error               { rc.record("close"); return nil }
func (rc *reasonReadCloser) CloseWithReason(reason string) error {
	rc.record(reason)
	return nil
}

func (rc *reasonReadCloser) record(reason string) {
	rc.mu.Lock()
	rc.closed = append(rc.closed, reason)
	rc.mu.Unlock()
	select {
	case rc.closedC <- struct{}{}:
	default:
	}
}

type execFakeRunner struct {
	stdout          io.ReadCloser
	process         processHandle
	stdin           io.WriteCloser
	waitErr         error
	waitDelay       time.Duration
	startErr        error
	stdoutErr       error
	stdinErr        error
	allowNilProcess bool
	started         atomic.Bool
}

func (f *execFakeRunner) Start() error {
	if f.startErr != nil {
		return f.startErr
	}
	f.started.Store(true)
	return nil
}
func (f *execFakeRunner) Wait() error {
	if f.waitDelay > 0 {
		time.Sleep(f.waitDelay)
	}
	return f.waitErr
}
func (f *execFakeRunner) StdoutPipe() (io.ReadCloser, error) {
	if f.stdoutErr != nil {
		return nil, f.stdoutErr
	}
	if f.stdout == nil {
		f.stdout = io.NopCloser(strings.NewReader(""))
	}
	return f.stdout, nil
}
func (f *execFakeRunner) StdinPipe() (io.WriteCloser, error) {
	if f.stdinErr != nil {
		return nil, f.stdinErr
	}
	if f.stdin != nil {
		return f.stdin, nil
	}
	return &writeCloserStub{}, nil
}
func (f *execFakeRunner) SetStderr(io.Writer) {}
func (f *execFakeRunner) SetDir(string)      {}
func (f *execFakeRunner) Process() processHandle {
	if f.process != nil {
		return f.process
	}
	if f.allowNilProcess {
		return nil
	}
	return &execFakeProcess{pid: 1}
}

func TestExecutorHelperCoverage(t *testing.T) {
	t.Run("realCmdAndProcess", func(t *testing.T) {
		rc := &realCmd{}
		if err := rc.Start(); err == nil {
			t.Fatalf("expected error for nil command")
		}
		if err := rc.Wait(); err == nil {
			t.Fatalf("expected error for nil command")
		}
		if _, err := rc.StdoutPipe(); err == nil {
			t.Fatalf("expected error for nil command")
		}
		if _, err := rc.StdinPipe(); err == nil {
			t.Fatalf("expected error for nil command")
		}
		rc.SetStderr(io.Discard)
		if rc.Process() != nil {
			t.Fatalf("expected nil process")
		}
		rcWithCmd := &realCmd{cmd: &exec.Cmd{}}
		rcWithCmd.SetStderr(io.Discard)
		echoCmd := exec.Command("echo", "ok")
		rcProc := &realCmd{cmd: echoCmd}
		stdoutPipe, err := rcProc.StdoutPipe()
		if err != nil {
			t.Fatalf("StdoutPipe error: %v", err)
		}
		stdinPipe, err := rcProc.StdinPipe()
		if err != nil {
			t.Fatalf("StdinPipe error: %v", err)
		}
		rcProc.SetStderr(io.Discard)
		if err := rcProc.Start(); err != nil {
			t.Fatalf("Start failed: %v", err)
		}
		_, _ = stdinPipe.Write([]byte{})
		_ = stdinPipe.Close()
		procHandle := rcProc.Process()
		if procHandle == nil {
			t.Fatalf("expected process handle")
		}
		_ = procHandle.Signal(syscall.SIGTERM)
		_ = procHandle.Kill()
		_ = rcProc.Wait()
		_, _ = io.ReadAll(stdoutPipe)

		rp := &realProcess{}
		if rp.Pid() != 0 {
			t.Fatalf("nil process should have pid 0")
		}
		if rp.Kill() != nil {
			t.Fatalf("nil process Kill should be nil")
		}
		if rp.Signal(syscall.SIGTERM) != nil {
			t.Fatalf("nil process Signal should be nil")
		}
		rpLive := &realProcess{proc: &os.Process{Pid: 99}}
		if rpLive.Pid() != 99 {
			t.Fatalf("expected pid 99, got %d", rpLive.Pid())
		}
		_ = rpLive.Kill()
		_ = rpLive.Signal(syscall.SIGTERM)
	})

	t.Run("topologicalSortAndSkip", func(t *testing.T) {
		layers, err := topologicalSort([]TaskSpec{{ID: "root"}, {ID: "child", Dependencies: []string{"root"}}})
		if err != nil || len(layers) != 2 {
			t.Fatalf("unexpected topological sort result: layers=%d err=%v", len(layers), err)
		}
		if _, err := topologicalSort([]TaskSpec{{ID: "cycle", Dependencies: []string{"cycle"}}}); err == nil {
			t.Fatalf("expected cycle detection error")
		}

		failed := map[string]TaskResult{"root": {ExitCode: 1}}
		if skip, _ := shouldSkipTask(TaskSpec{ID: "child", Dependencies: []string{"root"}}, failed); !skip {
			t.Fatalf("should skip when dependency failed")
		}
		if skip, _ := shouldSkipTask(TaskSpec{ID: "leaf"}, failed); skip {
			t.Fatalf("should not skip task without dependencies")
		}
		if skip, _ := shouldSkipTask(TaskSpec{ID: "child-ok", Dependencies: []string{"root"}}, map[string]TaskResult{}); skip {
			t.Fatalf("should not skip when dependencies succeeded")
		}
	})

	t.Run("cancelledTaskResult", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		res := cancelledTaskResult("t1", ctx)
		if res.ExitCode != 130 {
			t.Fatalf("expected cancel exit code, got %d", res.ExitCode)
		}

		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 0)
		defer timeoutCancel()
		res = cancelledTaskResult("t2", timeoutCtx)
		if res.ExitCode != 124 {
			t.Fatalf("expected timeout exit code, got %d", res.ExitCode)
		}
	})

	t.Run("generateFinalOutputAndArgs", func(t *testing.T) {
		out := generateFinalOutput([]TaskResult{
			{TaskID: "ok", ExitCode: 0},
			{TaskID: "fail", ExitCode: 1, Error: "boom"},
		})
		if !strings.Contains(out, "ok") || !strings.Contains(out, "fail") {
			t.Fatalf("unexpected summary output: %s", out)
		}
		out = generateFinalOutput([]TaskResult{{TaskID: "rich", ExitCode: 0, SessionID: "sess", LogPath: "/tmp/log", Message: "hello"}})
		if !strings.Contains(out, "Session: sess") || !strings.Contains(out, "Log: /tmp/log") || !strings.Contains(out, "hello") {
			t.Fatalf("rich output missing fields: %s", out)
		}

		args := buildCodexArgs(&Config{Mode: "new", WorkDir: "/tmp"}, "task")
		if len(args) == 0 || args[3] != "/tmp" {
			t.Fatalf("unexpected codex args: %+v", args)
		}
		args = buildCodexArgs(&Config{Mode: "resume", SessionID: "sess"}, "target")
		if args[3] != "resume" || args[4] != "sess" {
			t.Fatalf("unexpected resume args: %+v", args)
		}
	})

	t.Run("executeConcurrentWrapper", func(t *testing.T) {
		orig := runCodexTaskFn
		defer func() { runCodexTaskFn = orig }()
		runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
			return TaskResult{TaskID: task.ID, ExitCode: 0, Message: "done"}
		}
		os.Setenv("CODEAGENT_MAX_PARALLEL_WORKERS", "1")
		defer os.Unsetenv("CODEAGENT_MAX_PARALLEL_WORKERS")

		results := executeConcurrent([][]TaskSpec{{{ID: "wrap"}}}, 1)
		if len(results) != 1 || results[0].TaskID != "wrap" {
			t.Fatalf("unexpected wrapper results: %+v", results)
		}

		unbounded := executeConcurrentWithContext(context.Background(), [][]TaskSpec{{{ID: "unbounded"}}}, 1, 0)
		if len(unbounded) != 1 || unbounded[0].ExitCode != 0 {
			t.Fatalf("unexpected unbounded result: %+v", unbounded)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cancelled := executeConcurrentWithContext(ctx, [][]TaskSpec{{{ID: "cancel"}}}, 1, 1)
		if cancelled[0].ExitCode == 0 {
			t.Fatalf("expected cancelled result, got %+v", cancelled[0])
		}
	})
}

func TestExecutorRunCodexTaskWithContext(t *testing.T) {
	origRunner := newCommandRunner
	defer func() { newCommandRunner = origRunner }()

	t.Run("success", func(t *testing.T) {
		var firstStdout *reasonReadCloser
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			rc := newReasonReadCloser(`{"type":"item.completed","item":{"type":"agent_message","text":"hello"}}`)
			if firstStdout == nil {
				firstStdout = rc
			}
			return &execFakeRunner{stdout: rc, process: &execFakeProcess{pid: 1234}}
		}

		res := runCodexTaskWithContext(context.Background(), TaskSpec{ID: "task-1", Task: "payload", WorkDir: "."}, nil, nil, false, false, 1)
		if res.Error != "" || res.Message != "hello" || res.ExitCode != 0 {
			t.Fatalf("unexpected result: %+v", res)
		}

		select {
		case <-firstStdout.closedC:
		case <-time.After(1 * time.Second):
			t.Fatalf("stdout not closed with reason")
		}

		orig := runCodexTaskFn
		runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
			return TaskResult{TaskID: task.ID, ExitCode: 0, Message: "ok"}
		}
		t.Cleanup(func() { runCodexTaskFn = orig })

		if res := runCodexTask(TaskSpec{Task: "task-text", WorkDir: "."}, true, 1); res.ExitCode != 0 {
			t.Fatalf("runCodexTask failed: %+v", res)
		}

		msg, threadID, code := runCodexProcess(context.Background(), []string{"arg"}, "content", false, 1)
		if code != 0 || msg == "" {
			t.Fatalf("runCodexProcess unexpected result: msg=%q code=%d threadID=%s", msg, code, threadID)
		}
	})

	t.Run("startErrors", func(t *testing.T) {
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{startErr: errors.New("executable file not found"), process: &execFakeProcess{pid: 1}}
		}
		res := runCodexTaskWithContext(context.Background(), TaskSpec{Task: "payload", WorkDir: "."}, nil, nil, false, false, 1)
		if res.ExitCode != 127 {
			t.Fatalf("expected missing executable exit code, got %d", res.ExitCode)
		}

		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{startErr: errors.New("start failed"), process: &execFakeProcess{pid: 2}}
		}
		res = runCodexTaskWithContext(context.Background(), TaskSpec{Task: "payload", WorkDir: "."}, nil, nil, false, false, 1)
		if res.ExitCode == 0 {
			t.Fatalf("expected non-zero exit on start failure")
		}
	})

	t.Run("timeoutAndPipes", func(t *testing.T) {
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{
				stdout:    newReasonReadCloser(`{"type":"item.completed","item":{"type":"agent_message","text":"slow"}}`),
				process:   &execFakeProcess{pid: 5},
				waitDelay: 20 * time.Millisecond,
			}
		}
		res := runCodexTaskWithContext(context.Background(), TaskSpec{Task: "payload", WorkDir: ".", UseStdin: true}, nil, nil, false, false, 0)
		if res.ExitCode == 0 {
			t.Fatalf("expected timeout result, got %+v", res)
		}
	})

	t.Run("pipeErrors", func(t *testing.T) {
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{stdoutErr: errors.New("stdout fail"), process: &execFakeProcess{pid: 6}}
		}
		res := runCodexTaskWithContext(context.Background(), TaskSpec{Task: "payload", WorkDir: "."}, nil, nil, false, false, 1)
		if res.ExitCode == 0 {
			t.Fatalf("expected failure on stdout pipe error")
		}

		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{stdinErr: errors.New("stdin fail"), process: &execFakeProcess{pid: 7}}
		}
		res = runCodexTaskWithContext(context.Background(), TaskSpec{Task: "payload", WorkDir: ".", UseStdin: true}, nil, nil, false, false, 1)
		if res.ExitCode == 0 {
			t.Fatalf("expected failure on stdin pipe error")
		}
	})

	t.Run("waitExitError", func(t *testing.T) {
		err := exec.Command("false").Run()
		exitErr, _ := err.(*exec.ExitError)
		if exitErr == nil {
			t.Fatalf("expected exec.ExitError")
		}
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{
				stdout:  newReasonReadCloser(`{"type":"item.completed","item":{"type":"agent_message","text":"ignored"}}`),
				process: &execFakeProcess{pid: 8},
				waitErr: exitErr,
			}
		}
		res := runCodexTaskWithContext(context.Background(), TaskSpec{Task: "payload", WorkDir: "."}, nil, nil, false, false, 1)
		if res.ExitCode == 0 {
			t.Fatalf("expected non-zero exit on wait error")
		}
	})

	t.Run("contextCancelled", func(t *testing.T) {
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{
				stdout:    newReasonReadCloser(`{"type":"item.completed","item":{"type":"agent_message","text":"cancel"}}`),
				process:   &execFakeProcess{pid: 9},
				waitDelay: 10 * time.Millisecond,
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		res := runCodexTaskWithContext(ctx, TaskSpec{Task: "payload", WorkDir: "."}, nil, nil, false, false, 1)
		if res.ExitCode == 0 {
			t.Fatalf("expected cancellation result")
		}
	})

	t.Run("silentLogger", func(t *testing.T) {
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{
				stdout:  newReasonReadCloser(`{"type":"item.completed","item":{"type":"agent_message","text":"quiet"}}`),
				process: &execFakeProcess{pid: 10},
			}
		}
		_ = closeLogger()
		res := runCodexTaskWithContext(context.Background(), TaskSpec{Task: "payload", WorkDir: "."}, nil, nil, false, true, 1)
		if res.ExitCode != 0 || res.LogPath == "" {
			t.Fatalf("expected success with temp logger, got %+v", res)
		}
		_ = closeLogger()
	})

	t.Run("missingMessage", func(t *testing.T) {
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &execFakeRunner{
				stdout:  newReasonReadCloser(`{"type":"item.completed","item":{"type":"task","text":"noop"}}`),
				process: &execFakeProcess{pid: 11},
			}
		}
		res := runCodexTaskWithContext(context.Background(), TaskSpec{Task: "payload", WorkDir: "."}, nil, nil, false, false, 1)
		if res.ExitCode == 0 {
			t.Fatalf("expected failure when no agent_message returned")
		}
	})
}

func TestExecutorSignalAndTermination(t *testing.T) {
	forceKillDelay.Store(0)
	defer forceKillDelay.Store(5)

	proc := &execFakeProcess{pid: 42}
	cmd := &execFakeRunner{process: proc}

	origNotify := signalNotifyFn
	origStop := signalStopFn
	defer func() {
		signalNotifyFn = origNotify
		signalStopFn = origStop
	}()

	signalNotifyFn = func(c chan<- os.Signal, sigs ...os.Signal) {
		go func() { c <- syscall.SIGINT }()
	}
	signalStopFn = func(c chan<- os.Signal) {}

	forwardSignals(context.Background(), cmd, func(string) {})
	time.Sleep(20 * time.Millisecond)

	proc.mu.Lock()
	signalled := len(proc.signals)
	proc.mu.Unlock()
	if signalled == 0 {
		t.Fatalf("process did not receive signal")
	}
	if proc.killed.Load() == 0 {
		t.Fatalf("process was not killed after signal")
	}

	timer := terminateProcess(cmd)
	if timer == nil {
		t.Fatalf("terminateProcess returned nil timer")
	}
	timer.Stop()

	ft := terminateCommand(cmd)
	if ft == nil {
		t.Fatalf("terminateCommand returned nil")
	}
	ft.Stop()

	cmdKill := &execFakeRunner{process: &execFakeProcess{pid: 50}}
	ftKill := terminateCommand(cmdKill)
	time.Sleep(10 * time.Millisecond)
	if p, ok := cmdKill.process.(*execFakeProcess); ok && p.killed.Load() == 0 {
		t.Fatalf("terminateCommand did not kill process")
	}
	ftKill.Stop()

	cmdKill2 := &execFakeRunner{process: &execFakeProcess{pid: 51}}
	timer2 := terminateProcess(cmdKill2)
	time.Sleep(10 * time.Millisecond)
	if p, ok := cmdKill2.process.(*execFakeProcess); ok && p.killed.Load() == 0 {
		t.Fatalf("terminateProcess did not kill process")
	}
	timer2.Stop()

	if terminateCommand(nil) != nil {
		t.Fatalf("terminateCommand should return nil for nil cmd")
	}
	if terminateCommand(&execFakeRunner{allowNilProcess: true}) != nil {
		t.Fatalf("terminateCommand should return nil when process is nil")
	}
	if terminateProcess(nil) != nil {
		t.Fatalf("terminateProcess should return nil for nil cmd")
	}
	if terminateProcess(&execFakeRunner{allowNilProcess: true}) != nil {
		t.Fatalf("terminateProcess should return nil when process is nil")
	}

	signalNotifyFn = func(c chan<- os.Signal, sigs ...os.Signal) {}
	ctxDone, cancelDone := context.WithCancel(context.Background())
	cancelDone()
	forwardSignals(ctxDone, &execFakeRunner{process: &execFakeProcess{pid: 70}}, func(string) {})
}

func TestExecutorCancelReasonAndCloseWithReason(t *testing.T) {
	if reason := cancelReason("", nil); !strings.Contains(reason, "Context") {
		t.Fatalf("unexpected cancelReason for nil ctx: %s", reason)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	if !strings.Contains(cancelReason("cmd", ctx), "timeout") {
		t.Fatalf("expected timeout reason")
	}
	cancelCtx, cancelFn := context.WithCancel(context.Background())
	cancelFn()
	if !strings.Contains(cancelReason("cmd", cancelCtx), "Execution cancelled") {
		t.Fatalf("expected cancellation reason")
	}
	if !strings.Contains(cancelReason("", cancelCtx), "codex") {
		t.Fatalf("expected default command name in cancel reason")
	}

	rc := &reasonReadCloser{r: strings.NewReader("data"), closedC: make(chan struct{}, 1)}
	closeWithReason(rc, "why")
	select {
	case <-rc.closedC:
	default:
		t.Fatalf("CloseWithReason was not called")
	}

	plain := io.NopCloser(strings.NewReader("x"))
	closeWithReason(plain, "noop")
	closeWithReason(nil, "noop")
}

func TestExecutorForceKillTimerStop(t *testing.T) {
	done := make(chan struct{}, 1)
	ft := &forceKillTimer{timer: time.AfterFunc(50*time.Millisecond, func() { done <- struct{}{} }), done: done}
	ft.Stop()

	done2 := make(chan struct{}, 1)
	ft2 := &forceKillTimer{timer: time.AfterFunc(0, func() { done2 <- struct{}{} }), done: done2}
	time.Sleep(10 * time.Millisecond)
	ft2.Stop()

	var nilTimer *forceKillTimer
	nilTimer.Stop()
	(&forceKillTimer{}).Stop()
}

func TestExecutorForwardSignalsDefaults(t *testing.T) {
	origNotify := signalNotifyFn
	origStop := signalStopFn
	signalNotifyFn = nil
	signalStopFn = nil
	defer func() {
		signalNotifyFn = origNotify
		signalStopFn = origStop
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	forwardSignals(ctx, &execFakeRunner{process: &execFakeProcess{pid: 80}}, func(string) {})
	time.Sleep(10 * time.Millisecond)
}
