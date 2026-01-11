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
	c.buildAliases()
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
	out.Aliases = append(out.Aliases, overlay.Aliases...)
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
	if overlay.OnlyOn != nil {
		only := *overlay.OnlyOn
		only.Branches = append([]string(nil), overlay.OnlyOn.Branches...)
		only.Tags = append([]string(nil), overlay.OnlyOn.Tags...)
		out.OnlyOn = &only
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
	if overlay.Jitter != "" {
		out.Jitter = overlay.Jitter
	}
	if overlay.Retries > 0 {
		out.Retries = overlay.Retries
	}
	if overlay.MaxRetries > 0 {
		out.MaxRetries = overlay.MaxRetries
	}
	if len(overlay.RetryOnExitCodes) > 0 {
		out.RetryOnExitCodes = append([]int(nil), overlay.RetryOnExitCodes...)
	}
	if len(overlay.RetryOnRegex) > 0 {
		out.RetryOnRegex = append([]string(nil), overlay.RetryOnRegex...)
	}
	if len(overlay.RetryOnSignal) > 0 {
		out.RetryOnSignal = append([]string(nil), overlay.RetryOnSignal...)
	}
	if overlay.ContinueOnError {
		out.ContinueOnError = true
	}
	if overlay.AllowFailure {
		out.AllowFailure = true
	}
	if overlay.Priority != 0 {
		out.Priority = overlay.Priority
	}
	if overlay.Group != "" {
		out.Group = overlay.Group
	}
	if overlay.RunDir != "" {
		out.RunDir = overlay.RunDir
	}
	if overlay.Seed != 0 {
		out.Seed = overlay.Seed
	}
	out.SeedEnv = mergeStringMap(out.SeedEnv, overlay.SeedEnv)
	if overlay.Confirm != "" {
		out.Confirm = overlay.Confirm
	}
	if overlay.Isolate {
		out.Isolate = true
	}
	if overlay.Silent {
		out.Silent = true
	}
	if overlay.IfMissing {
		out.IfMissing = true
	}
	if overlay.Limits != nil {
		limits := *overlay.Limits
		out.Limits = &limits
	}
	if overlay.Resources != nil {
		res := *overlay.Resources
		out.Resources = &res
	}
	if overlay.Remote != nil {
		remote := *overlay.Remote
		out.Remote = &remote
	}
	if overlay.Scheduler != nil {
		scheduler := *overlay.Scheduler
		scheduler.Args = append([]string(nil), overlay.Scheduler.Args...)
		out.Scheduler = &scheduler
	}
	out.Inputs = append(out.Inputs, overlay.Inputs...)
	out.Outputs = append(out.Outputs, overlay.Outputs...)
	out.OutputPaths = append(out.OutputPaths, overlay.OutputPaths...)
	out.Output = mergeStringMap(out.Output, overlay.Output)
	if overlay.Capture != nil {
		capture := *overlay.Capture
		out.Capture = &capture
	}
	out.Exports = mergeStringMap(out.Exports, overlay.Exports)
	out.Watch = append(out.Watch, overlay.Watch...)
	out.Artifacts = append(out.Artifacts, overlay.Artifacts...)
	out.Tags = append(out.Tags, overlay.Tags...)
	out.Secrets = append(out.Secrets, overlay.Secrets...)
	out.Require = append(out.Require, overlay.Require...)
	if len(overlay.Datasets) > 0 {
		out.Datasets = append(out.Datasets, overlay.Datasets...)
	}
	if len(overlay.DatasetOutputs) > 0 {
		out.DatasetOutputs = append(out.DatasetOutputs, overlay.DatasetOutputs...)
	}
	if overlay.Split != nil {
		out.Split = cloneSplitSpec(overlay.Split)
	}
	if overlay.Validate != nil {
		out.Validate = cloneValidateSpec(overlay.Validate)
	}
	if overlay.Stats != nil {
		out.Stats = cloneStatsSpec(overlay.Stats)
	}
	if overlay.Metrics != nil {
		out.Metrics = cloneMetricsSpec(overlay.Metrics)
	}
	if overlay.Canary != nil {
		out.Canary = cloneCanarySpec(overlay.Canary)
	}
	if overlay.Benchmark != nil {
		out.Benchmark = cloneBenchmarkSpec(overlay.Benchmark)
	}
	if overlay.Experiment != nil {
		out.Experiment = cloneExperimentSpec(overlay.Experiment)
	}
	if overlay.Snapshot != nil {
		out.Snapshot = cloneSnapshotSpec(overlay.Snapshot)
	}
	if overlay.Sign != nil {
		out.Sign = cloneSignSpec(overlay.Sign)
	}
	if overlay.SBOM != nil {
		out.SBOM = cloneSBOMSpec(overlay.SBOM)
	}
	if overlay.Checkpoint != nil {
		out.Checkpoint = cloneCheckpointSpec(overlay.Checkpoint)
	}
	if overlay.ModelCard != nil {
		out.ModelCard = cloneModelCardSpec(overlay.ModelCard)
	}
	if overlay.Notebook != nil {
		out.Notebook = cloneNotebookSpec(overlay.Notebook)
	}
	if overlay.Export != nil {
		out.Export = cloneExportSpec(overlay.Export)
	}
	if overlay.Offline != nil {
		out.Offline = cloneOfflineSpec(overlay.Offline)
	}
	if overlay.Matrix != nil {
		out.Matrix = mergeMatrix(out.Matrix, overlay.Matrix)
	}
	if overlay.Sweep != nil {
		out.Sweep = cloneSweepSpec(overlay.Sweep)
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
