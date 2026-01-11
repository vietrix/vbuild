package runner

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func (r *Runner) ExportEnv(taskName string, path string) error {
	if taskName == "" {
		taskName = "default"
	}
	if path == "" {
		path = ".env"
	}
	plan, err := r.buildPlan([]string{taskName})
	if err != nil {
		return err
	}
	task, ok := plan.tasks[taskName]
	if !ok || task == nil {
		return fmt.Errorf("unknown task: %s", taskName)
	}
	vars := r.taskVars(plan, taskName, task)
	env := r.taskEnv(vars, task)
	return writeEnvFile(r.resolvePath(path), env)
}

func (r *Runner) PrintVars(taskName string, out io.Writer) error {
	if taskName == "" {
		taskName = "default"
	}
	if out == nil {
		out = os.Stdout
	}
	plan, err := r.buildPlan([]string{taskName})
	if err != nil {
		return err
	}
	task, ok := plan.tasks[taskName]
	if !ok || task == nil {
		return fmt.Errorf("unknown task: %s", taskName)
	}
	vars := r.taskVars(plan, taskName, task)
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(out, "%s=%s\n", key, vars[key])
	}
	return nil
}

func writeEnvFile(path string, values map[string]string) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	builder := strings.Builder{}
	for _, key := range keys {
		value := values[key]
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(escapeEnvValue(value))
		builder.WriteString("\n")
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func escapeEnvValue(value string) string {
	if value == "" {
		return "\"\""
	}
	if strings.ContainsAny(value, " \t\n\r\"'") {
		escaped := strings.ReplaceAll(value, "\"", "\\\"")
		return "\"" + escaped + "\""
	}
	return value
}
