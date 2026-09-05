package output

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	domaintask "codeagent-wrapper/internal/domain/task"

	"github.com/goccy/go-json"
)

type Summary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type Payload struct {
	Results []domaintask.TaskResult `json:"results"`
	Summary Summary                 `json:"summary"`
}

type CommandOutput struct {
	Stdout   string
	ExitCode int
	LogError string
}

func RenderSingleTaskBanner(w io.Writer, name string, backend string, command []string, pid int, logPath string) {
	if w == nil {
		return
	}

	fmt.Fprintf(w, "[%s]\n", name)
	fmt.Fprintf(w, "  Backend: %s\n", backend)
	fmt.Fprintf(w, "  Command: %s\n", strings.Join(command, " "))
	fmt.Fprintf(w, "  PID: %d\n", pid)
	fmt.Fprintf(w, "  Log: %s\n", logPath)
}

func FinalizeSingleTask(backend string, outputPath string, result domaintask.TaskResult) (CommandOutput, error) {
	exitCode := result.ExitCode
	if exitCode == 0 && strings.TrimSpace(result.Message) == "" {
		errMsg := fmt.Sprintf("no output message: backend=%s returned empty result.Message with exit_code=0", backend)
		exitCode = 1
		if strings.TrimSpace(result.Error) == "" {
			result.Error = errMsg
		}
	}

	if err := WriteStructuredOutput(outputPath, []domaintask.TaskResult{result}); err != nil {
		return CommandOutput{}, err
	}

	return CommandOutput{
		Stdout:   renderSingleTaskMessage(result),
		ExitCode: exitCode,
		LogError: result.Error,
	}, nil
}

func FinalizeParallel(stdout string, exitCode int) CommandOutput {
	return CommandOutput{
		Stdout:   stdout,
		ExitCode: exitCode,
	}
}

func Emit(w io.Writer, output CommandOutput) {
	if w == nil || output.Stdout == "" {
		return
	}
	fmt.Fprintln(w, output.Stdout)
}

func WriteStructuredOutput(path string, results []domaintask.TaskResult) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	cleanPath := filepath.Clean(path)
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory for %q: %w", cleanPath, err)
	}

	f, err := os.Create(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %q: %w", cleanPath, err)
	}

	encodeErr := json.NewEncoder(f).Encode(Payload{
		Results: results,
		Summary: summarizeResults(results),
	})
	closeErr := f.Close()

	if encodeErr != nil {
		return fmt.Errorf("failed to write structured output to %q: %w", cleanPath, encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close output file %q: %w", cleanPath, closeErr)
	}
	return nil
}

func summarizeResults(results []domaintask.TaskResult) Summary {
	summary := Summary{Total: len(results)}
	for _, res := range results {
		if res.ExitCode == 0 && res.Error == "" {
			summary.Success++
		} else {
			summary.Failed++
		}
	}
	return summary
}

func renderSingleTaskMessage(result domaintask.TaskResult) string {
	if strings.TrimSpace(result.Message) == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString(result.Message)
	if result.SessionID != "" {
		b.WriteString("\n\n---\nSESSION_ID: ")
		b.WriteString(result.SessionID)
	}
	return b.String()
}
