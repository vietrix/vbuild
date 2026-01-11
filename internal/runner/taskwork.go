package runner

import "github.com/vietrix/vbuild/internal/config"

func taskHasWork(task *config.Task, commands []string) bool {
	if task == nil {
		return false
	}
	if len(commands) > 0 || len(task.Pre) > 0 || len(task.Post) > 0 {
		return true
	}
	if task.Split != nil || task.Validate != nil || task.Stats != nil || task.Notebook != nil {
		return true
	}
	if task.Export != nil || task.Benchmark != nil || task.Canary != nil || task.Metrics != nil {
		return true
	}
	if len(task.DatasetOutputs) > 0 || task.Checkpoint != nil || task.ModelCard != nil {
		return true
	}
	if task.SBOM != nil || task.Sign != nil || task.Snapshot != nil {
		return true
	}
	return false
}
