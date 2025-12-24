package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type integrationSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type integrationOutput struct {
	Results []TaskResult       `json:"results"`
	Summary integrationSummary `json:"summary"`
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func parseIntegrationOutput(t *testing.T, out string) integrationOutput {
	t.Helper()
	var payload integrationOutput

	lines := strings.Split(out, "\n")
	var currentTask *TaskResult
	inTaskResults := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse new format header: "X tasks | Y passed | Z failed"
		if strings.Contains(line, "tasks |") && strings.Contains(line, "passed |") {
			parts := strings.Split(line, "|")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasSuffix(p, "tasks") {
					fmt.Sscanf(p, "%d tasks", &payload.Summary.Total)
				} else if strings.HasSuffix(p, "passed") {
					fmt.Sscanf(p, "%d passed", &payload.Summary.Success)
				} else if strings.HasSuffix(p, "failed") {
					fmt.Sscanf(p, "%d failed", &payload.Summary.Failed)
				}
			}
		} else if strings.HasPrefix(line, "Total:") {
			// Legacy format: "Total: X | Success: Y | Failed: Z"
			parts := strings.Split(line, "|")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "Total:") {
					fmt.Sscanf(p, "Total: %d", &payload.Summary.Total)
				} else if strings.HasPrefix(p, "Success:") {
					fmt.Sscanf(p, "Success: %d", &payload.Summary.Success)
				} else if strings.HasPrefix(p, "Failed:") {
					fmt.Sscanf(p, "Failed: %d", &payload.Summary.Failed)
				}
			}
		} else if line == "## Task Results" {
			inTaskResults = true
		} else if line == "## Summary" {
			// End of task results section
			if currentTask != nil {
				payload.Results = append(payload.Results, *currentTask)
				currentTask = nil
			}
			inTaskResults = false
		} else if inTaskResults && strings.HasPrefix(line, "### ") {
			// New task: ### task-id ✓ 92% or ### task-id PASS 92% (ASCII mode)
			if currentTask != nil {
				payload.Results = append(payload.Results, *currentTask)
			}
			currentTask = &TaskResult{}

			taskLine := strings.TrimPrefix(line, "### ")
			success, warning, failed := getStatusSymbols()
			// Parse different formats
			if strings.Contains(taskLine, " "+success) {
				parts := strings.Split(taskLine, " "+success)
				currentTask.TaskID = strings.TrimSpace(parts[0])
				currentTask.ExitCode = 0
				// Extract coverage if present
				if len(parts) > 1 {
					coveragePart := strings.TrimSpace(parts[1])
					if strings.HasSuffix(coveragePart, "%") {
						currentTask.Coverage = coveragePart
					}
				}
			} else if strings.Contains(taskLine, " "+warning) {
				parts := strings.Split(taskLine, " "+warning)
				currentTask.TaskID = strings.TrimSpace(parts[0])
				currentTask.ExitCode = 0
			} else if strings.Contains(taskLine, " "+failed) {
				parts := strings.Split(taskLine, " "+failed)
				currentTask.TaskID = strings.TrimSpace(parts[0])
				currentTask.ExitCode = 1
			} else {
				currentTask.TaskID = taskLine
			}
		} else if currentTask != nil && inTaskResults {
			// Parse task details
			if strings.HasPrefix(line, "Exit code:") {
				fmt.Sscanf(line, "Exit code: %d", &currentTask.ExitCode)
			} else if strings.HasPrefix(line, "Error:") {
				currentTask.Error = strings.TrimPrefix(line, "Error: ")
			} else if strings.HasPrefix(line, "Log:") {
				currentTask.LogPath = strings.TrimSpace(strings.TrimPrefix(line, "Log:"))
			} else if strings.HasPrefix(line, "Did:") {
				currentTask.KeyOutput = strings.TrimSpace(strings.TrimPrefix(line, "Did:"))
			} else if strings.HasPrefix(line, "Detail:") {
				// Error detail for failed tasks
				if currentTask.Message == "" {
					currentTask.Message = strings.TrimSpace(strings.TrimPrefix(line, "Detail:"))
				}
			}
		} else if strings.HasPrefix(line, "--- Task:") {
			// Legacy full output format
			if currentTask != nil {
				payload.Results = append(payload.Results, *currentTask)
			}
			currentTask = &TaskResult{}
			currentTask.TaskID = strings.TrimSuffix(strings.TrimPrefix(line, "--- Task: "), " ---")
		} else if currentTask != nil && !inTaskResults {
			// Legacy format parsing
			if strings.HasPrefix(line, "Status: SUCCESS") {
				currentTask.ExitCode = 0
			} else if strings.HasPrefix(line, "Status: FAILED") {
				if strings.Contains(line, "exit code") {
					fmt.Sscanf(line, "Status: FAILED (exit code %d)", &currentTask.ExitCode)
				} else {
					currentTask.ExitCode = 1
				}
			} else if strings.HasPrefix(line, "Error:") {
				currentTask.Error = strings.TrimPrefix(line, "Error: ")
			} else if strings.HasPrefix(line, "Session:") {
				currentTask.SessionID = strings.TrimPrefix(line, "Session: ")
			} else if strings.HasPrefix(line, "Log:") {
				currentTask.LogPath = strings.TrimSpace(strings.TrimPrefix(line, "Log:"))
			}
		}
	}

	// Handle last task
	if currentTask != nil {
		payload.Results = append(payload.Results, *currentTask)
	}

	return payload
}

func extractTaskBlock(t *testing.T, output, taskID string) string {
	t.Helper()
	header := fmt.Sprintf("--- Task: %s ---", taskID)
	lines := strings.Split(output, "\n")
	var block []string
	collecting := false
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if !collecting {
			if trimmed == header {
				collecting = true
				block = append(block, trimmed)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "--- Task: ") && trimmed != header {
			break
		}
		block = append(block, trimmed)
	}
	if len(block) == 0 {
		t.Fatalf("task block %s not found in output:\n%s", taskID, output)
	}
	return strings.Join(block, "\n")
}

func findResultByID(t *testing.T, payload integrationOutput, id string) TaskResult {
	t.Helper()
	for _, res := range payload.Results {
		if res.TaskID == id {
			return res
		}
	}
	t.Fatalf("result for task %s not found", id)
	return TaskResult{}
}

func TestRunParallelEndToEnd_OrderAndConcurrency(t *testing.T) {
	defer resetTestHooks()
	origRun := runCodexTaskFn
	t.Cleanup(func() {
		runCodexTaskFn = origRun
		resetTestHooks()
	})

	input := `---TASK---
id: A
---CONTENT---
task-a
---TASK---
id: B
dependencies: A
---CONTENT---
task-b
---TASK---
id: C
dependencies: B
---CONTENT---
task-c
---TASK---
id: D
---CONTENT---
task-d
---TASK---
id: E
---CONTENT---
task-e`
	stdinReader = bytes.NewReader([]byte(input))
	os.Args = []string{"codeagent-wrapper", "--parallel"}

	var mu sync.Mutex
	starts := make(map[string]time.Time)
	ends := make(map[string]time.Time)
	var running int64
	var maxParallel int64

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		start := time.Now()
		mu.Lock()
		starts[task.ID] = start
		mu.Unlock()

		cur := atomic.AddInt64(&running, 1)
		for {
			prev := atomic.LoadInt64(&maxParallel)
			if cur <= prev {
				break
			}
			if atomic.CompareAndSwapInt64(&maxParallel, prev, cur) {
				break
			}
		}

		time.Sleep(40 * time.Millisecond)

		mu.Lock()
		ends[task.ID] = time.Now()
		mu.Unlock()

		atomic.AddInt64(&running, -1)
		return TaskResult{TaskID: task.ID, ExitCode: 0, Message: task.Task}
	}

	var exitCode int
	output := captureStdout(t, func() {
		exitCode = run()
	})

	if exitCode != 0 {
		t.Fatalf("run() exit = %d, want 0", exitCode)
	}

	payload := parseIntegrationOutput(t, output)
	if payload.Summary.Failed != 0 || payload.Summary.Total != 5 || payload.Summary.Success != 5 {
		t.Fatalf("unexpected summary: %+v", payload.Summary)
	}

	aEnd := ends["A"]
	bStart := starts["B"]
	cStart := starts["C"]
	bEnd := ends["B"]
	if aEnd.IsZero() || bStart.IsZero() || bEnd.IsZero() || cStart.IsZero() {
		t.Fatalf("missing timestamps, starts=%v ends=%v", starts, ends)
	}
	if !aEnd.Before(bStart) && !aEnd.Equal(bStart) {
		t.Fatalf("B should start after A ends: A_end=%v B_start=%v", aEnd, bStart)
	}
	if !bEnd.Before(cStart) && !bEnd.Equal(cStart) {
		t.Fatalf("C should start after B ends: B_end=%v C_start=%v", bEnd, cStart)
	}

	dStart := starts["D"]
	eStart := starts["E"]
	if dStart.IsZero() || eStart.IsZero() {
		t.Fatalf("missing D/E start times: %v", starts)
	}
	delta := dStart.Sub(eStart)
	if delta < 0 {
		delta = -delta
	}
	if delta > 25*time.Millisecond {
		t.Fatalf("D and E should run in parallel, delta=%v", delta)
	}
	if maxParallel < 2 {
		t.Fatalf("expected at least 2 concurrent tasks, got %d", maxParallel)
	}
}

func TestRunParallelCycleDetectionStopsExecution(t *testing.T) {
	defer resetTestHooks()
	origRun := runCodexTaskFn
	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		t.Fatalf("task %s should not execute on cycle", task.ID)
		return TaskResult{}
	}
	t.Cleanup(func() {
		runCodexTaskFn = origRun
		resetTestHooks()
	})

	input := `---TASK---
id: A
dependencies: B
---CONTENT---
a
---TASK---
id: B
dependencies: A
---CONTENT---
b`
	stdinReader = bytes.NewReader([]byte(input))
	os.Args = []string{"codeagent-wrapper", "--parallel"}

	exitCode := 0
	output := captureStdout(t, func() {
		exitCode = run()
	})

	if exitCode == 0 {
		t.Fatalf("cycle should cause non-zero exit, got %d", exitCode)
	}
	if strings.TrimSpace(output) != "" {
		t.Fatalf("expected no JSON output on cycle, got %q", output)
	}
}

func TestRunParallelOutputsIncludeLogPaths(t *testing.T) {
	defer resetTestHooks()
	origRun := runCodexTaskFn
	t.Cleanup(func() {
		runCodexTaskFn = origRun
		resetTestHooks()
	})

	tempDir := t.TempDir()
	logPathFor := func(id string) string {
		return filepath.Join(tempDir, fmt.Sprintf("%s.log", id))
	}

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		res := TaskResult{
			TaskID:    task.ID,
			Message:   fmt.Sprintf("result-%s", task.ID),
			SessionID: fmt.Sprintf("session-%s", task.ID),
			LogPath:   logPathFor(task.ID),
		}
		if task.ID == "beta" {
			res.ExitCode = 9
			res.Error = "boom"
		}
		return res
	}

	input := `---TASK---
id: alpha
---CONTENT---
task-alpha
---TASK---
id: beta
---CONTENT---
task-beta`
	stdinReader = bytes.NewReader([]byte(input))
	os.Args = []string{"codex-wrapper", "--parallel"}

	var exitCode int
	output := captureStdout(t, func() {
		exitCode = run()
	})

	if exitCode != 9 {
		t.Fatalf("parallel run exit=%d, want 9", exitCode)
	}

	payload := parseIntegrationOutput(t, output)
	alpha := findResultByID(t, payload, "alpha")
	beta := findResultByID(t, payload, "beta")

	if alpha.LogPath != logPathFor("alpha") {
		t.Fatalf("alpha log path = %q, want %q", alpha.LogPath, logPathFor("alpha"))
	}
	if beta.LogPath != logPathFor("beta") {
		t.Fatalf("beta log path = %q, want %q", beta.LogPath, logPathFor("beta"))
	}

	for _, id := range []string{"alpha", "beta"} {
		// Summary mode shows log paths in table format, not "Log: xxx"
		logPath := logPathFor(id)
		if !strings.Contains(output, logPath) {
			t.Fatalf("parallel output missing log path %q for %s:\n%s", logPath, id, output)
		}
	}
}

func TestRunParallelStartupLogsPrinted(t *testing.T) {
	defer resetTestHooks()

	tempDir := setTempDirEnv(t, t.TempDir())
	input := `---TASK---
id: a
---CONTENT---
fail
---TASK---
id: b
---CONTENT---
ok-b
---TASK---
id: c
dependencies: a
---CONTENT---
should-skip
---TASK---
id: d
---CONTENT---
ok-d`
	stdinReader = bytes.NewReader([]byte(input))
	os.Args = []string{"codex-wrapper", "--parallel"}

	expectedLog := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))

	origRun := runCodexTaskFn
	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		path := expectedLog
		if logger := activeLogger(); logger != nil && logger.Path() != "" {
			path = logger.Path()
		}
		if task.ID == "a" {
			return TaskResult{TaskID: task.ID, ExitCode: 3, Error: "boom", LogPath: path}
		}
		return TaskResult{TaskID: task.ID, ExitCode: 0, Message: task.Task, LogPath: path}
	}
	t.Cleanup(func() { runCodexTaskFn = origRun })

	var exitCode int
	var stdoutOut string
	stderrOut := captureStderr(t, func() {
		stdoutOut = captureStdout(t, func() {
			exitCode = run()
		})
	})

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit due to task failure, got %d", exitCode)
	}
	if stdoutOut == "" {
		t.Fatalf("expected parallel summary on stdout")
	}

	lines := strings.Split(strings.TrimSpace(stderrOut), "\n")
	var bannerSeen bool
	var taskLines []string
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if line == "=== Starting Parallel Execution ===" {
			if bannerSeen {
				t.Fatalf("banner printed multiple times:\n%s", stderrOut)
			}
			bannerSeen = true
			continue
		}
		taskLines = append(taskLines, line)
	}

	if !bannerSeen {
		t.Fatalf("expected startup banner in stderr, got:\n%s", stderrOut)
	}

	// After parallel log isolation fix, each task has its own log file
	expectedLines := map[string]struct{}{
		fmt.Sprintf("Task a: Log: %s", filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d-a.log", os.Getpid()))): {},
		fmt.Sprintf("Task b: Log: %s", filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d-b.log", os.Getpid()))): {},
		fmt.Sprintf("Task d: Log: %s", filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d-d.log", os.Getpid()))): {},
	}

	if len(taskLines) != len(expectedLines) {
		t.Fatalf("startup log lines mismatch, got %d lines:\n%s", len(taskLines), stderrOut)
	}

	for _, line := range taskLines {
		if _, ok := expectedLines[line]; !ok {
			t.Fatalf("unexpected startup line %q\nstderr:\n%s", line, stderrOut)
		}
	}
}

func TestRunNonParallelOutputsIncludeLogPathsIntegration(t *testing.T) {
	defer resetTestHooks()

	tempDir := setTempDirEnv(t, t.TempDir())
	os.Args = []string{"codex-wrapper", "integration-log-check"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	codexCommand = "echo"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string {
		return []string{`{"type":"thread.started","thread_id":"integration-session"}` + "\n" + `{"type":"item.completed","item":{"type":"agent_message","text":"done"}}`}
	}

	var exitCode int
	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			exitCode = run()
		})
	})

	if exitCode != 0 {
		t.Fatalf("run() exit=%d, want 0", exitCode)
	}
	expectedLog := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))
	wantLine := fmt.Sprintf("Log: %s", expectedLog)
	if !strings.Contains(stderr, wantLine) {
		t.Fatalf("stderr missing %q, got: %q", wantLine, stderr)
	}
}

func TestRunParallelPartialFailureBlocksDependents(t *testing.T) {
	defer resetTestHooks()
	origRun := runCodexTaskFn
	t.Cleanup(func() {
		runCodexTaskFn = origRun
		resetTestHooks()
	})

	tempDir := t.TempDir()
	logPathFor := func(id string) string {
		return filepath.Join(tempDir, fmt.Sprintf("%s.log", id))
	}

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		path := logPathFor(task.ID)
		if task.ID == "A" {
			return TaskResult{TaskID: "A", ExitCode: 2, Error: "boom", LogPath: path}
		}
		return TaskResult{TaskID: task.ID, ExitCode: 0, Message: task.Task, LogPath: path}
	}

	input := `---TASK---
id: A
---CONTENT---
fail
---TASK---
id: B
dependencies: A
---CONTENT---
blocked
---TASK---
id: D
---CONTENT---
ok-d
---TASK---
id: E
---CONTENT---
ok-e`
	stdinReader = bytes.NewReader([]byte(input))
	os.Args = []string{"codeagent-wrapper", "--parallel"}

	var exitCode int
	output := captureStdout(t, func() {
		exitCode = run()
	})

	payload := parseIntegrationOutput(t, output)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when a task fails, got %d", exitCode)
	}

	resA := findResultByID(t, payload, "A")
	resB := findResultByID(t, payload, "B")
	resD := findResultByID(t, payload, "D")
	resE := findResultByID(t, payload, "E")

	if resA.ExitCode == 0 {
		t.Fatalf("task A should fail, got %+v", resA)
	}
	if resB.ExitCode == 0 || !strings.Contains(resB.Error, "dependencies") {
		t.Fatalf("task B should be skipped due to dependency failure, got %+v", resB)
	}
	if resD.ExitCode != 0 || resE.ExitCode != 0 {
		t.Fatalf("independent tasks should run successfully, D=%+v E=%+v", resD, resE)
	}
	if payload.Summary.Failed != 2 || payload.Summary.Total != 4 {
		t.Fatalf("unexpected summary after partial failure: %+v", payload.Summary)
	}
	if resA.LogPath != logPathFor("A") {
		t.Fatalf("task A log path = %q, want %q", resA.LogPath, logPathFor("A"))
	}
	if resB.LogPath != "" {
		t.Fatalf("task B should not report a log path when skipped, got %q", resB.LogPath)
	}
	if resD.LogPath != logPathFor("D") || resE.LogPath != logPathFor("E") {
		t.Fatalf("expected log paths for D/E, got D=%q E=%q", resD.LogPath, resE.LogPath)
	}
	// Summary mode shows log paths in table, verify they appear in output
	for _, id := range []string{"A", "D", "E"} {
		logPath := logPathFor(id)
		if !strings.Contains(output, logPath) {
			t.Fatalf("task %s log path %q not found in output:\n%s", id, logPath, output)
		}
	}
	// Task B was skipped, should have "-" or empty log path in table
	if resB.LogPath != "" {
		t.Fatalf("skipped task B should have empty log path, got %q", resB.LogPath)
	}
}

func TestRunParallelTimeoutPropagation(t *testing.T) {
	defer resetTestHooks()
	origRun := runCodexTaskFn
	t.Cleanup(func() {
		runCodexTaskFn = origRun
		resetTestHooks()
		os.Unsetenv("CODEX_TIMEOUT")
	})

	var receivedTimeout int
	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		receivedTimeout = timeout
		return TaskResult{TaskID: task.ID, ExitCode: 124, Error: "timeout"}
	}

	os.Setenv("CODEX_TIMEOUT", "1")
	input := `---TASK---
id: T
---CONTENT---
slow`
	stdinReader = bytes.NewReader([]byte(input))
	os.Args = []string{"codeagent-wrapper", "--parallel"}

	exitCode := 0
	output := captureStdout(t, func() {
		exitCode = run()
	})

	payload := parseIntegrationOutput(t, output)
	if receivedTimeout != 1 {
		t.Fatalf("expected timeout 1s to propagate, got %d", receivedTimeout)
	}
	if exitCode != 124 {
		t.Fatalf("expected timeout exit code 124, got %d", exitCode)
	}
	if payload.Summary.Failed != 1 || payload.Summary.Total != 1 {
		t.Fatalf("unexpected summary for timeout case: %+v", payload.Summary)
	}
	res := findResultByID(t, payload, "T")
	if res.Error == "" || res.ExitCode != 124 {
		t.Fatalf("timeout result not propagated, got %+v", res)
	}
}

func TestRunConcurrentSpeedupBenchmark(t *testing.T) {
	defer resetTestHooks()
	origRun := runCodexTaskFn
	t.Cleanup(func() {
		runCodexTaskFn = origRun
		resetTestHooks()
	})

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		time.Sleep(50 * time.Millisecond)
		return TaskResult{TaskID: task.ID}
	}

	tasks := make([]TaskSpec, 10)
	for i := range tasks {
		tasks[i] = TaskSpec{ID: fmt.Sprintf("task-%d", i)}
	}
	layers := [][]TaskSpec{tasks}

	serialStart := time.Now()
	for _, task := range tasks {
		_ = runCodexTaskFn(task, 5)
	}
	serialElapsed := time.Since(serialStart)

	concurrentStart := time.Now()
	_ = executeConcurrent(layers, 5)
	concurrentElapsed := time.Since(concurrentStart)

	if concurrentElapsed >= serialElapsed/5 {
		t.Fatalf("expected concurrent time <20%% of serial, serial=%v concurrent=%v", serialElapsed, concurrentElapsed)
	}
	ratio := float64(concurrentElapsed) / float64(serialElapsed)
	t.Logf("speedup ratio (concurrent/serial)=%.3f", ratio)
}

func TestRunStartupCleanupRemovesOrphansEndToEnd(t *testing.T) {
	defer resetTestHooks()

	tempDir := setTempDirEnv(t, t.TempDir())

	orphanA := createTempLog(t, tempDir, "codex-wrapper-5001.log")
	orphanB := createTempLog(t, tempDir, "codex-wrapper-5002-extra.log")
	orphanC := createTempLog(t, tempDir, "codex-wrapper-5003-suffix.log")
	runningPID := 81234
	runningLog := createTempLog(t, tempDir, fmt.Sprintf("codex-wrapper-%d.log", runningPID))
	unrelated := createTempLog(t, tempDir, "wrapper.log")

	stubProcessRunning(t, func(pid int) bool {
		return pid == runningPID || pid == os.Getpid()
	})
	stubProcessStartTime(t, func(pid int) time.Time {
		if pid == runningPID || pid == os.Getpid() {
			return time.Now().Add(-1 * time.Hour)
		}
		return time.Time{}
	})

	codexCommand = createFakeCodexScript(t, "tid-startup", "ok")
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	os.Args = []string{"codex-wrapper", "task"}

	if exit := run(); exit != 0 {
		t.Fatalf("run() exit=%d, want 0", exit)
	}

	for _, orphan := range []string{orphanA, orphanB, orphanC} {
		if _, err := os.Stat(orphan); !os.IsNotExist(err) {
			t.Fatalf("expected orphan %s to be removed, err=%v", orphan, err)
		}
	}
	if _, err := os.Stat(runningLog); err != nil {
		t.Fatalf("expected running log to remain, err=%v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("expected unrelated file to remain, err=%v", err)
	}
}

func TestRunStartupCleanupConcurrentWrappers(t *testing.T) {
	defer resetTestHooks()

	tempDir := setTempDirEnv(t, t.TempDir())

	const totalLogs = 40
	for i := 0; i < totalLogs; i++ {
		createTempLog(t, tempDir, fmt.Sprintf("codex-wrapper-%d.log", 9000+i))
	}

	stubProcessRunning(t, func(pid int) bool {
		return false
	})
	stubProcessStartTime(t, func(int) time.Time { return time.Time{} })

	var wg sync.WaitGroup
	const instances = 5
	start := make(chan struct{})

	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			runStartupCleanup()
		}()
	}

	close(start)
	wg.Wait()

	matches, err := filepath.Glob(filepath.Join(tempDir, "codex-wrapper-*.log"))
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected all orphan logs to be removed, remaining=%v", matches)
	}
}

func TestRunCleanupFlagEndToEnd_Success(t *testing.T) {
	defer resetTestHooks()

	tempDir := setTempDirEnv(t, t.TempDir())

	staleA := createTempLog(t, tempDir, "codex-wrapper-2100.log")
	staleB := createTempLog(t, tempDir, "codex-wrapper-2200-extra.log")
	keeper := createTempLog(t, tempDir, "codex-wrapper-2300.log")

	stubProcessRunning(t, func(pid int) bool {
		return pid == 2300 || pid == os.Getpid()
	})
	stubProcessStartTime(t, func(pid int) time.Time {
		if pid == 2300 || pid == os.Getpid() {
			return time.Now().Add(-1 * time.Hour)
		}
		return time.Time{}
	})

	os.Args = []string{"codex-wrapper", "--cleanup"}

	var exitCode int
	output := captureStdout(t, func() {
		exitCode = run()
	})

	if exitCode != 0 {
		t.Fatalf("cleanup exit = %d, want 0", exitCode)
	}

	// Check that output contains expected counts and file names
	if !strings.Contains(output, "Cleanup completed") {
		t.Fatalf("missing 'Cleanup completed' in output: %q", output)
	}
	if !strings.Contains(output, "Files scanned: 3") {
		t.Fatalf("missing 'Files scanned: 3' in output: %q", output)
	}
	if !strings.Contains(output, "Files deleted: 2") {
		t.Fatalf("missing 'Files deleted: 2' in output: %q", output)
	}
	if !strings.Contains(output, "Files kept: 1") {
		t.Fatalf("missing 'Files kept: 1' in output: %q", output)
	}
	if !strings.Contains(output, "codex-wrapper-2100.log") || !strings.Contains(output, "codex-wrapper-2200-extra.log") {
		t.Fatalf("missing deleted file names in output: %q", output)
	}
	if !strings.Contains(output, "codex-wrapper-2300.log") {
		t.Fatalf("missing kept file names in output: %q", output)
	}

	for _, path := range []string{staleA, staleB} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, err=%v", path, err)
		}
	}
	if _, err := os.Stat(keeper); err != nil {
		t.Fatalf("expected kept log to remain, err=%v", err)
	}

	currentLog := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))
	if _, err := os.Stat(currentLog); err == nil {
		t.Fatalf("cleanup mode should not create new log file %s", currentLog)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat(%s) unexpected error: %v", currentLog, err)
	}
}

func TestRunCleanupFlagEndToEnd_FailureDoesNotAffectStartup(t *testing.T) {
	defer resetTestHooks()

	tempDir := setTempDirEnv(t, t.TempDir())

	calls := 0
	cleanupLogsFn = func() (CleanupStats, error) {
		calls++
		return CleanupStats{Scanned: 1}, fmt.Errorf("permission denied")
	}

	os.Args = []string{"codex-wrapper", "--cleanup"}

	var exitCode int
	errOutput := captureStderr(t, func() {
		exitCode = run()
	})

	if exitCode != 1 {
		t.Fatalf("cleanup failure exit = %d, want 1", exitCode)
	}
	if !strings.Contains(errOutput, "Cleanup failed") || !strings.Contains(errOutput, "permission denied") {
		t.Fatalf("cleanup stderr = %q, want failure message", errOutput)
	}
	if calls != 1 {
		t.Fatalf("cleanup called %d times, want 1", calls)
	}

	currentLog := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))
	if _, err := os.Stat(currentLog); err == nil {
		t.Fatalf("cleanup failure should not create new log file %s", currentLog)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat(%s) unexpected error: %v", currentLog, err)
	}

	cleanupLogsFn = func() (CleanupStats, error) {
		return CleanupStats{}, nil
	}
	codexCommand = createFakeCodexScript(t, "tid-cleanup-e2e", "ok")
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	os.Args = []string{"codex-wrapper", "post-cleanup task"}

	var normalExit int
	normalOutput := captureStdout(t, func() {
		normalExit = run()
	})

	if normalExit != 0 {
		t.Fatalf("normal run exit = %d, want 0", normalExit)
	}
	if !strings.Contains(normalOutput, "ok") {
		t.Fatalf("normal run output = %q, want codex output", normalOutput)
	}
}
