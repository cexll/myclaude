package cli

import (
	"fmt"
	"io"

	appoutput "codeagent-wrapper/internal/application/output"
	domaintask "codeagent-wrapper/internal/domain/task"
)

func WriteParallelModeArgumentError(w io.Writer, name string) {
	if w == nil {
		return
	}
	fmt.Fprintln(w, "ERROR: --parallel reads its task configuration from stdin; no positional arguments are allowed.")
	fmt.Fprintln(w, "Usage examples:")
	fmt.Fprintf(w, "  %s --parallel < tasks.txt\n", name)
	fmt.Fprintf(w, "  echo '...' | %s --parallel\n", name)
	fmt.Fprintf(w, "  %s --parallel <<'EOF'\n", name)
	fmt.Fprintf(w, "  %s --parallel --full-output <<'EOF'  # include full task output\n", name)
}

func WriteParallelModeFlagError(w io.Writer) {
	if w == nil {
		return
	}
	fmt.Fprintln(w, "ERROR: --parallel reads its task configuration from stdin; only --backend, --model, --output, --full-output and --skip-permissions are allowed.")
}

func EmitParallelResult(w io.Writer, stdout string, exitCode int) int {
	appoutput.Emit(w, appoutput.FinalizeParallel(stdout, exitCode))
	return exitCode
}

func RenderSingleTaskBanner(w io.Writer, name string, backend string, command []string, pid int, logPath string) {
	appoutput.RenderSingleTaskBanner(w, name, backend, command, pid, logPath)
}

func EmitSingleTaskResult(w io.Writer, backend string, outputPath string, result domaintask.TaskResult) (appoutput.CommandOutput, error) {
	output, err := appoutput.FinalizeSingleTask(backend, outputPath, result)
	if err != nil {
		return appoutput.CommandOutput{}, err
	}
	appoutput.Emit(w, output)
	return output, nil
}
