package task

import (
	"strings"
	"testing"
)

func TestTopologicalSort_BranchingGraph(t *testing.T) {
	tasks := []TaskSpec{
		{ID: "root"},
		{ID: "left", Dependencies: []string{"root"}},
		{ID: "right", Dependencies: []string{"root"}},
		{ID: "leaf", Dependencies: []string{"left", "right"}},
	}

	layers, err := TopologicalSort(tasks)
	if err != nil {
		t.Fatalf("TopologicalSort() error = %v", err)
	}
	if len(layers) != 3 || len(layers[1]) != 2 {
		t.Fatalf("TopologicalSort() = %#v, want 3 layers with 2 tasks in layer 2", layers)
	}
}

func TestTopologicalSort_CycleDetection(t *testing.T) {
	tasks := []TaskSpec{
		{ID: "a", Dependencies: []string{"b"}},
		{ID: "b", Dependencies: []string{"a"}},
	}

	_, err := TopologicalSort(tasks)
	if err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("TopologicalSort() error = %v, want cycle detected", err)
	}
}
