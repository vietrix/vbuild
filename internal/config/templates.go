package config

import (
	"fmt"
	"os"
	"strings"
)

func (c *Config) applyEnvOverrides() {
	const prefix = "VBUILD_VAR_"
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, prefix) {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimPrefix(parts[0], prefix)
		if strings.TrimSpace(key) == "" {
			continue
		}
		c.Vars[key] = parts[1]
	}
}

func (c *Config) resolveTemplates() error {
	if len(c.Templates) == 0 {
		return nil
	}
	for name, task := range c.Tasks {
		if task == nil || task.Use == "" {
			continue
		}
		template, ok := c.Templates[task.Use]
		if !ok || template == nil {
			return fmt.Errorf("tasks.%s.use refers to unknown template %q", name, task.Use)
		}
		merged := mergeTask(template, task)
		merged.Use = task.Use
		c.Tasks[name] = merged
	}
	c.normalizeTasks()
	return nil
}

func mergeTask(base, overlay *Task) *Task {
	out := cloneTask(base)
	if out == nil {
		out = &Task{}
	}
	if overlay == nil {
		return out
	}

	if overlay.Desc != "" {
		out.Desc = overlay.Desc
	}
	out.Deps = append(out.Deps, overlay.Deps...)
	out.Needs = append(out.Needs, overlay.Needs...)
	out.DependsOn = append(out.DependsOn, overlay.DependsOn...)
	out.Run = append(out.Run, overlay.Run...)
	out.Pre = append(out.Pre, overlay.Pre...)
	out.Post = append(out.Post, overlay.Post...)
	out.Env = mergeStringMap(out.Env, overlay.Env)
	out.Vars = mergeStringMap(out.Vars, overlay.Vars)
	out.Vars = mergeStringMap(out.Vars, overlay.With)
	if overlay.Parallel {
		out.Parallel = true
	}
	if overlay.Fanout {
		out.Fanout = true
	}
	if overlay.Workdir != "" {
		out.Workdir = overlay.Workdir
	}
	if overlay.Shell != "" {
		out.Shell = overlay.Shell
	}
	if overlay.When != "" {
		out.When = overlay.When
	}
	if overlay.Cache != "" {
		out.Cache = overlay.Cache
	}
	if overlay.Timeout != "" {
		out.Timeout = overlay.Timeout
	}
	if overlay.Backoff != "" {
		out.Backoff = overlay.Backoff
	}
	if overlay.Retries > 0 {
		out.Retries = overlay.Retries
	}
	if len(overlay.RetryOnExitCodes) > 0 {
		out.RetryOnExitCodes = append([]int(nil), overlay.RetryOnExitCodes...)
	}
	if len(overlay.RetryOnRegex) > 0 {
		out.RetryOnRegex = append([]string(nil), overlay.RetryOnRegex...)
	}
	if overlay.ContinueOnError {
		out.ContinueOnError = true
	}
	if overlay.AllowFailure {
		out.AllowFailure = true
	}
	if overlay.Confirm != "" {
		out.Confirm = overlay.Confirm
	}
	if overlay.Isolate {
		out.Isolate = true
	}
	if overlay.Limits != nil {
		limits := *overlay.Limits
		out.Limits = &limits
	}
	if overlay.Remote != nil {
		remote := *overlay.Remote
		out.Remote = &remote
	}
	out.Inputs = append(out.Inputs, overlay.Inputs...)
	out.Outputs = append(out.Outputs, overlay.Outputs...)
	out.OutputPaths = append(out.OutputPaths, overlay.OutputPaths...)
	out.Exports = mergeStringMap(out.Exports, overlay.Exports)
	out.Watch = append(out.Watch, overlay.Watch...)
	out.Artifacts = append(out.Artifacts, overlay.Artifacts...)
	out.Tags = append(out.Tags, overlay.Tags...)
	out.Secrets = append(out.Secrets, overlay.Secrets...)
	if overlay.Matrix != nil {
		out.Matrix = mergeMatrix(out.Matrix, overlay.Matrix)
	}
	if overlay.Docker != nil {
		out.Docker = cloneDocker(overlay.Docker)
	}
	return out
}

func mergeMatrix(base, overlay map[string][]string) map[string][]string {
	if base == nil && overlay == nil {
		return map[string][]string{}
	}
	out := map[string][]string{}
	for key, values := range base {
		out[key] = append([]string(nil), values...)
	}
	for key, values := range overlay {
		out[key] = append([]string(nil), values...)
	}
	return out
}
