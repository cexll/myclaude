package cli

import (
	"bytes"
	"strings"
	"testing"

	domaintask "codeagent-wrapper/internal/domain/task"
)

func TestWriteParallelModeArgumentError(t *testing.T) {
	var stderr bytes.Buffer
	WriteParallelModeArgumentError(&stderr, "codeagent-wrapper")
	got := stderr.String()
	if !strings.Contains(got, "reads its task configuration from stdin") || !strings.Contains(got, "codeagent-wrapper --parallel") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestEmitSingleTaskResult(t *testing.T) {
	var stdout bytes.Buffer
	output, err := EmitSingleTaskResult(&stdout, "codex", "", domaintask.TaskResult{
		ExitCode:  0,
		Message:   "ok",
		SessionID: "sess-1",
	})
	if err != nil {
		t.Fatalf("EmitSingleTaskResult() error = %v", err)
	}
	if output.ExitCode != 0 || !strings.Contains(stdout.String(), "SESSION_ID: sess-1") {
		t.Fatalf("output=%#v stdout=%q", output, stdout.String())
	}
}
