package executor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type runtimeTestProcess struct {
	mu      sync.Mutex
	signals []os.Signal
	killed  bool
	pid     int
	onStop  func(error)
}

func (p *runtimeTestProcess) Pid() int {
	if p.pid == 0 {
		return 4242
	}
	return p.pid
}

func (p *runtimeTestProcess) Kill() error {
	p.mu.Lock()
	p.killed = true
	onStop := p.onStop
	p.mu.Unlock()
	if onStop != nil {
		onStop(errors.New("killed"))
	}
	return nil
}

func (p *runtimeTestProcess) Signal(sig os.Signal) error {
	p.mu.Lock()
	p.signals = append(p.signals, sig)
	onStop := p.onStop
	p.mu.Unlock()
	if onStop != nil {
		onStop(errors.New("signaled"))
	}
	return nil
}

func (p *runtimeTestProcess) signalCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.signals)
}

func (p *runtimeTestProcess) wasKilled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.killed
}

type runtimeTestCmd struct {
	stdoutR *io.PipeReader
	stdoutW *io.PipeWriter
	stderrR *io.PipeReader
	stderrW *io.PipeWriter
	waitCh  chan error
	proc    *runtimeTestProcess

	startOnce sync.Once
	waitOnce  sync.Once
	startFn   func(*runtimeTestCmd)
}

func newRuntimeTestCmd(startFn func(*runtimeTestCmd)) *runtimeTestCmd {
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	cmd := &runtimeTestCmd{
		stdoutR: stdoutR,
		stdoutW: stdoutW,
		stderrR: stderrR,
		stderrW: stderrW,
		waitCh:  make(chan error, 1),
		startFn: startFn,
	}
	cmd.proc = &runtimeTestProcess{
		onStop: func(err error) {
			cmd.finish(err)
		},
	}
	return cmd
}

func (c *runtimeTestCmd) Start() error {
	c.startOnce.Do(func() {
		if c.startFn != nil {
			c.startFn(c)
		}
	})
	return nil
}

func (c *runtimeTestCmd) Wait() error {
	return <-c.waitCh
}

func (c *runtimeTestCmd) StdoutPipe() (io.ReadCloser, error) {
	return c.stdoutR, nil
}

func (c *runtimeTestCmd) StderrPipe() (io.ReadCloser, error) {
	return c.stderrR, nil
}

func (c *runtimeTestCmd) StdinPipe() (io.WriteCloser, error) {
	return nil, errors.New("stdin not supported in test")
}

func (c *runtimeTestCmd) SetStderr(io.Writer) {}
func (c *runtimeTestCmd) SetDir(string)       {}
func (c *runtimeTestCmd) SetEnv(map[string]string) {
}
func (c *runtimeTestCmd) UnsetEnv(...string) {}

func (c *runtimeTestCmd) Process() processHandle {
	return c.proc
}

func (c *runtimeTestCmd) finish(err error) {
	c.waitOnce.Do(func() {
		_ = c.stdoutW.Close()
		_ = c.stderrW.Close()
		c.waitCh <- err
	})
}

func TestRunCodexTaskWithContext_IgnoresWrapperTimeoutAndWaitsForExit(t *testing.T) {
	var progress bytes.Buffer
	restoreProgress := SetProgressOutput(&progress)
	defer restoreProgress()
	restoreHeartbeat := SetProgressHeartbeatInterval(200 * time.Millisecond)
	defer restoreHeartbeat()

	restoreRunner := SetNewCommandRunner(func(ctx context.Context, name string, args ...string) CommandRunner {
		return newRuntimeTestCmd(func(cmd *runtimeTestCmd) {
			go func() {
				_, _ = io.WriteString(cmd.stdoutW, strings.Join([]string{
					`{"type":"thread.started","thread_id":"session-1"}`,
					`{"type":"item.completed","item":{"type":"agent_message","text":"done"}}`,
					`{"type":"turn.completed"}`,
				}, "\n")+"\n")
				time.Sleep(1500 * time.Millisecond)
				cmd.finish(nil)
			}()
		})
	})
	defer restoreRunner()

	start := time.Now()
	result := RunCodexTaskWithContext(
		context.Background(),
		TaskSpec{
			ID:      "task-1",
			Task:    "hi",
			WorkDir: ".",
		},
		nil,
		"codex",
		func(*Config, string) []string { return []string{"--json", "hi"} },
		nil,
		false,
		false,
		1,
	)
	elapsed := time.Since(start)

	if result.ExitCode != 0 {
		t.Fatalf("exit=%d error=%q", result.ExitCode, result.Error)
	}
	if result.Message != "done" {
		t.Fatalf("message=%q, want %q", result.Message, "done")
	}
	if result.SessionID != "session-1" {
		t.Fatalf("session_id=%q, want %q", result.SessionID, "session-1")
	}
	if elapsed < 1400*time.Millisecond {
		t.Fatalf("elapsed=%v, want >= 1.4s to prove wrapper timeout is ignored", elapsed)
	}
	got := progress.String()
	for _, want := range []string{"status=started", "status=streaming", "status=backend-complete", "status=running", "status=completed"} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress=%q, want %q", got, want)
		}
	}
}

func TestRunCodexTaskWithContext_CancelledContextTerminatesProcess(t *testing.T) {
	var cmd *runtimeTestCmd
	restoreRunner := SetNewCommandRunner(func(ctx context.Context, name string, args ...string) CommandRunner {
		cmd = newRuntimeTestCmd(func(cmd *runtimeTestCmd) {
			go func() {
				_, _ = io.WriteString(cmd.stdoutW, `{"type":"thread.started","thread_id":"session-cancel"}`+"\n")
			}()
		})
		return cmd
	})
	defer restoreRunner()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	result := RunCodexTaskWithContext(
		ctx,
		TaskSpec{
			ID:      "task-cancel",
			Task:    "hi",
			WorkDir: ".",
		},
		nil,
		"codex",
		func(*Config, string) []string { return []string{"--json", "hi"} },
		nil,
		false,
		true,
		1,
	)

	if result.ExitCode != 130 {
		t.Fatalf("exit=%d, want 130 (error=%q)", result.ExitCode, result.Error)
	}
	if !strings.Contains(result.Error, "execution cancelled") {
		t.Fatalf("error=%q, want cancellation message", result.Error)
	}
	if cmd == nil {
		t.Fatal("expected test command to be created")
	}
	if cmd.proc.signalCount() == 0 && !cmd.proc.wasKilled() {
		t.Fatal("expected cancellation to signal or kill the process")
	}
}

func TestCancelledTaskResult_DoesNotExposeTimeoutSemantics(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	result := cancelledTaskResult("task-deadline", ctx)

	if result.ExitCode != 130 {
		t.Fatalf("exit=%d, want 130", result.ExitCode)
	}
	if result.Error != "execution cancelled" {
		t.Fatalf("error=%q, want %q", result.Error, "execution cancelled")
	}
}
