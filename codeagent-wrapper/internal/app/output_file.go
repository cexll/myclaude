package wrapper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-json"
)

type outputSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type outputPayload struct {
	Results []TaskResult  `json:"results"`
	Summary outputSummary `json:"summary"`
}

func writeStructuredOutput(path string, results []TaskResult) error {
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

	encodeErr := json.NewEncoder(f).Encode(outputPayload{
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

func summarizeResults(results []TaskResult) outputSummary {
	summary := outputSummary{Total: len(results)}
	for _, res := range results {
		if res.ExitCode == 0 && res.Error == "" {
			summary.Success++
		} else {
			summary.Failed++
		}
	}
	return summary
}
