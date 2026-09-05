package wrapper

import (
	runtaskset "codeagent-wrapper/internal/application/runtaskset"
)

func parseParallelConfig(data []byte) (*runtaskset.ParallelConfig, error) {
	return runtaskset.ParseConfig(data)
}
