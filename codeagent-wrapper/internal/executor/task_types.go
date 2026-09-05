package executor

import (
	appruntask "codeagent-wrapper/internal/application/runtask"
	appruntaskset "codeagent-wrapper/internal/application/runtaskset"
	domaintask "codeagent-wrapper/internal/domain/task"
)

type ParallelConfig = appruntaskset.ParallelConfig
type TaskSpec = appruntask.Spec
type TaskResult = domaintask.TaskResult
