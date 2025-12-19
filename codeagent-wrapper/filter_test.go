package main

import (
	"bytes"
	"testing"
)

func TestFilteringWriter(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		input    string
		want     string
	}{
		{
			name:     "filter STARTUP lines",
			patterns: geminiNoisePatterns,
			input:    "[STARTUP] Recording metric\nHello World\n[STARTUP] Another line\n",
			want:     "Hello World\n",
		},
		{
			name:     "filter Warning lines",
			patterns: geminiNoisePatterns,
			input:    "Warning: something bad\nActual output\n",
			want:     "Actual output\n",
		},
		{
			name:     "filter multiple patterns",
			patterns: geminiNoisePatterns,
			input:    "YOLO mode is enabled\nSession cleanup disabled\nReal content\nLoading extension: foo\n",
			want:     "Real content\n",
		},
		{
			name:     "no filtering needed",
			patterns: geminiNoisePatterns,
			input:    "Line 1\nLine 2\nLine 3\n",
			want:     "Line 1\nLine 2\nLine 3\n",
		},
		{
			name:     "empty input",
			patterns: geminiNoisePatterns,
			input:    "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			fw := newFilteringWriter(&buf, tt.patterns)
			fw.Write([]byte(tt.input))
			fw.Flush()

			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFilteringWriterPartialLines(t *testing.T) {
	var buf bytes.Buffer
	fw := newFilteringWriter(&buf, geminiNoisePatterns)

	// Write partial line
	fw.Write([]byte("Hello "))
	fw.Write([]byte("World\n"))
	fw.Flush()

	if got := buf.String(); got != "Hello World\n" {
		t.Errorf("got %q, want %q", got, "Hello World\n")
	}
}
