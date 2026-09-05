package task

import (
	"fmt"
	"sort"
	"strings"
)

func TopologicalSort(tasks []TaskSpec) ([][]TaskSpec, error) {
	idToTask := make(map[string]TaskSpec, len(tasks))
	indegree := make(map[string]int, len(tasks))
	adj := make(map[string][]string, len(tasks))

	for _, task := range tasks {
		idToTask[task.ID] = task
		indegree[task.ID] = 0
	}

	for _, task := range tasks {
		for _, dep := range task.Dependencies {
			if _, ok := idToTask[dep]; !ok {
				return nil, fmt.Errorf("dependency %q not found for task %q", dep, task.ID)
			}
			indegree[task.ID]++
			adj[dep] = append(adj[dep], task.ID)
		}
	}

	queue := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if indegree[task.ID] == 0 {
			queue = append(queue, task.ID)
		}
	}

	layers := make([][]TaskSpec, 0)
	processed := 0

	for len(queue) > 0 {
		current := queue
		queue = nil
		layer := make([]TaskSpec, len(current))
		for i, id := range current {
			layer[i] = idToTask[id]
			processed++
		}
		layers = append(layers, layer)

		next := make([]string, 0)
		for _, id := range current {
			for _, neighbor := range adj[id] {
				indegree[neighbor]--
				if indegree[neighbor] == 0 {
					next = append(next, neighbor)
				}
			}
		}
		queue = append(queue, next...)
	}

	if processed != len(tasks) {
		cycleIDs := make([]string, 0)
		for id, deg := range indegree {
			if deg > 0 {
				cycleIDs = append(cycleIDs, id)
			}
		}
		sort.Strings(cycleIDs)
		return nil, fmt.Errorf("cycle detected involving tasks: %s", strings.Join(cycleIDs, ","))
	}

	return layers, nil
}
