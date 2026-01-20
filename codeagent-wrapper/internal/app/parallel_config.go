package wrapper

import (
	executor "codeagent-wrapper/internal/executor"
)

func parseParallelConfig(data []byte) (*ParallelConfig, error) {
	return executor.ParseParallelConfig(data)
}
