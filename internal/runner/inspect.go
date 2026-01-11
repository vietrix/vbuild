package runner

import (
	"encoding/json"
	"fmt"
	"io"
)

type inspectPayload struct {
	Name     string            `json:"name"`
	Base     string            `json:"base,omitempty"`
	Vars     map[string]string `json:"vars,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Deps     []string          `json:"deps,omitempty"`
	Commands []string          `json:"commands,omitempty"`
	Task     interface{}       `json:"task"`
}

func (r *Runner) Inspect(taskName string, out io.Writer) error {
	if out == nil {
		out = r.log.out
	}
	if taskName == "" {
		taskName = "default"
	}
	plan, err := r.buildPlan(r.allTaskNames())
	if err != nil {
		return err
	}
	task, ok := plan.tasks[taskName]
	if !ok || task == nil {
		return fmt.Errorf("unknown task: %s", taskName)
	}
	vars := r.taskVars(plan, taskName, task)
	env := r.taskEnv(vars, task)
	rawCommands := r.taskCommands(task, vars)
	commands := make([]string, 0, len(rawCommands))
	for _, cmd := range rawCommands {
		commands = append(commands, expandVars(cmd, vars))
	}
	payload := inspectPayload{
		Name:     taskName,
		Vars:     vars,
		Env:      env,
		Deps:     append([]string(nil), plan.deps[taskName]...),
		Commands: commands,
		Task:     task,
	}
	if variant, ok := plan.variants[taskName]; ok {
		payload.Base = variant.base
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
