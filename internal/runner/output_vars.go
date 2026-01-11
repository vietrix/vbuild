package runner

import (
	"os"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) recordOutputs(taskName string, task *config.Task, vars map[string]string) {
	if task == nil || len(task.Output) == 0 {
		return
	}
	values := map[string]string{}
	for key, value := range task.Output {
		values[key] = expandVars(value, vars)
	}
	r.outputsMu.Lock()
	r.outputs[taskName] = values
	r.outputsMu.Unlock()
}

func (r *Runner) outputsForDeps(plan *taskPlan, name string) map[string]string {
	out := map[string]string{}
	if plan == nil {
		return out
	}
	r.outputsMu.Lock()
	defer r.outputsMu.Unlock()
	for _, dep := range plan.deps[name] {
		values := r.outputs[dep]
		for key, value := range values {
			out[key] = value
		}
	}
	return out
}

func (r *Runner) outputsMissing(task *config.Task, vars map[string]string) (bool, error) {
	if task == nil {
		return true, nil
	}
	outputs, err := r.resolveOutputFiles(task, vars)
	if err != nil {
		return true, err
	}
	if len(outputs) == 0 {
		return true, nil
	}
	for _, path := range outputs {
		if _, err := os.Stat(path); err != nil {
			return true, nil
		}
	}
	return false, nil
}
