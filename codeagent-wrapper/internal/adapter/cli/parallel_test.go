package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type bufferWriter struct{ bytes.Buffer }

func TestExecuteParallelMode_PositionalArgsRejected(t *testing.T) {
	stdout := &bufferWriter{}
	stderr := &bufferWriter{}
	restoreStdout := stdoutWriter
	restoreStderr := stderrWriter
	stdoutWriter = func() io.Writer { return stdout }
	stderrWriter = func() io.Writer { return stderr }
	defer func() {
		stdoutWriter = restoreStdout
		stderrWriter = restoreStderr
	}()

	got := ExecuteParallelMode[int](&cobra.Command{}, []string{"bad"}, &Options{}, viper.New(), "wrapper", ParallelDeps[int]{})
	if got != 1 || !strings.Contains(stderr.String(), "no positional arguments") {
		t.Fatalf("got=%d stderr=%q", got, stderr.String())
	}
}

func TestExecuteParallelMode_BuildAndRun(t *testing.T) {
	stdout := &bufferWriter{}
	stderr := &bufferWriter{}
	restoreStdout := stdoutWriter
	restoreStderr := stderrWriter
	stdoutWriter = func() io.Writer { return stdout }
	stderrWriter = func() io.Writer { return stderr }
	defer func() {
		stdoutWriter = restoreStdout
		stderrWriter = restoreStderr
	}()

	cmd := &cobra.Command{}
	cmd.Flags().String("backend", "", "")
	opts := &Options{}
	got := ExecuteParallelMode(cmd, nil, opts, viper.New(), "wrapper", ParallelDeps[int]{
		BuildPlan: func(*cobra.Command, *Options, *viper.Viper) (int, error) { return 42, nil },
		RunPlan: func(plan int) (string, int, error) {
			if plan != 42 {
				t.Fatalf("plan=%d", plan)
			}
			return "summary", 7, nil
		},
	})
	if got != 7 || stdout.String() != "summary\n" || stderr.String() != "" {
		t.Fatalf("got=%d stdout=%q stderr=%q", got, stdout.String(), stderr.String())
	}
}

func TestExecuteParallelMode_RunError(t *testing.T) {
	stdout := &bufferWriter{}
	stderr := &bufferWriter{}
	restoreStdout := stdoutWriter
	restoreStderr := stderrWriter
	stdoutWriter = func() io.Writer { return stdout }
	stderrWriter = func() io.Writer { return stderr }
	defer func() {
		stdoutWriter = restoreStdout
		stderrWriter = restoreStderr
	}()

	got := ExecuteParallelMode[int](&cobra.Command{}, nil, &Options{}, viper.New(), "wrapper", ParallelDeps[int]{
		BuildPlan: func(*cobra.Command, *Options, *viper.Viper) (int, error) { return 1, nil },
		RunPlan:   func(int) (string, int, error) { return "", 0, errors.New("boom") },
	})
	if got != 1 || !strings.Contains(stderr.String(), "ERROR: boom") {
		t.Fatalf("got=%d stderr=%q", got, stderr.String())
	}
}
