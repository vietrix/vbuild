package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	sources := map[string][]byte{}
	cfg, err := loadRecursive(path, map[string]bool{}, sources)
	if err != nil {
		return nil, err
	}

	if isURL(path) {
		cfg.Path = path
	} else if abs, err := filepath.Abs(path); err == nil {
		cfg.Path = abs
	} else {
		cfg.Path = path
	}
	cfg.Hash = configHash(sources)
	cfg.Sources = sortedKeys(sources)

	cfg.normalize()
	cfg.applyEnvOverrides()
	if err := cfg.resolveTemplates(); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadRecursive(path string, visited map[string]bool, sources map[string][]byte) (*Config, error) {
	autoIncludes := []string{}
	if !isURL(path) {
		autoIncludes = append(autoIncludes, autoIncludeDir(path)...)
	}

	key, err := canonicalKey(path)
	if err != nil {
		return nil, err
	}
	if visited[key] {
		return nil, fmt.Errorf("config include cycle detected: %s", key)
	}
	visited[key] = true

	data, err := readConfigSource(path)
	if err != nil {
		return nil, err
	}
	if _, ok := sources[key]; !ok {
		sources[key] = data
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	cfg.normalize()

	base := &Config{}
	base.normalize()

	allIncludes := append([]string{}, autoIncludes...)
	allIncludes = append(allIncludes, cfg.Include...)
	expanded, err := expandIncludes(path, allIncludes)
	if err != nil {
		return nil, err
	}
	for _, inc := range expanded {
		resolved, err := resolveInclude(path, inc)
		if err != nil {
			return nil, err
		}
		child, err := loadRecursive(resolved, visited, sources)
		if err != nil {
			return nil, err
		}
		base = mergeConfigs(base, child)
	}

	merged := mergeConfigs(base, &cfg)
	merged.normalize()
	return merged, nil
}

func readConfigSource(path string) ([]byte, error) {
	if isURL(path) {
		req, err := http.NewRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("User-Agent", "vbuild")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch config: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch config: %s", resp.Status)
		}
		return ioReadAll(resp.Body)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return data, nil
}

func resolveInclude(basePath, include string) (string, error) {
	if isURL(include) {
		return include, nil
	}
	if isURL(basePath) {
		baseURL, err := url.Parse(basePath)
		if err != nil {
			return "", fmt.Errorf("parse base url: %w", err)
		}
		ref, err := url.Parse(include)
		if err != nil {
			return "", fmt.Errorf("parse include url: %w", err)
		}
		return baseURL.ResolveReference(ref).String(), nil
	}
	if filepath.IsAbs(include) {
		return include, nil
	}
	dir := filepath.Dir(basePath)
	return filepath.Join(dir, include), nil
}

func isURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func canonicalKey(path string) (string, error) {
	if isURL(path) {
		return path, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return abs, nil
}

func mergeConfigs(base, overlay *Config) *Config {
	out := &Config{}
	out.normalize()

	if base != nil {
		out.Workflow = base.Workflow
		out.EnvFile = base.EnvFile
		out.ArtifactsDir = base.ArtifactsDir
		out.Timeout = base.Timeout
		out.Seed = base.Seed
		out.SeedEnv = mergeStringMap(out.SeedEnv, base.SeedEnv)
		out.Offline = cloneOfflineSpec(base.Offline)
		out.Resources = cloneResourcePool(base.Resources)
		out.Datasets = mergeDatasetMap(out.Datasets, base.Datasets)
		out.Experiments = mergeExperimentDefaults(out.Experiments, base.Experiments)
		out.Registry = cloneRegistrySpec(base.Registry)
		out.Snapshot = cloneSnapshotSpec(base.Snapshot)
		out.Defaults = mergeDefaults(out.Defaults, base.Defaults)
		out.FailFast = base.FailFast
		out.CacheRemote = base.CacheRemote
		out.Artifacts = base.Artifacts
		out.Vars = mergeStringMap(out.Vars, base.Vars)
		out.Env = mergeStringMap(out.Env, base.Env)
		out.Tasks = mergeTaskMap(out.Tasks, base.Tasks)
		out.Templates = mergeTaskMap(out.Templates, base.Templates)
		out.Plugins = append(out.Plugins, base.Plugins...)
		out.LogPlugins = append(out.LogPlugins, base.LogPlugins...)
		out.Secrets = append(out.Secrets, base.Secrets...)
	}

	if overlay != nil {
		if overlay.Workflow != "" {
			out.Workflow = overlay.Workflow
		}
		if overlay.EnvFile != "" {
			out.EnvFile = overlay.EnvFile
		}
		if overlay.ArtifactsDir != "" {
			out.ArtifactsDir = overlay.ArtifactsDir
		}
		if overlay.Timeout != "" {
			out.Timeout = overlay.Timeout
		}
		if overlay.Seed != 0 {
			out.Seed = overlay.Seed
		}
		out.SeedEnv = mergeStringMap(out.SeedEnv, overlay.SeedEnv)
		if overlay.Offline != nil {
			out.Offline = mergeOfflineSpec(out.Offline, overlay.Offline)
		}
		if overlay.Resources != nil {
			out.Resources = mergeResourcePool(out.Resources, overlay.Resources)
		}
		if overlay.Datasets != nil {
			out.Datasets = mergeDatasetMap(out.Datasets, overlay.Datasets)
		}
		out.Experiments = mergeExperimentDefaults(out.Experiments, overlay.Experiments)
		if overlay.Registry != nil {
			out.Registry = cloneRegistrySpec(overlay.Registry)
		}
		if overlay.Snapshot != nil {
			out.Snapshot = cloneSnapshotSpec(overlay.Snapshot)
		}
		out.Defaults = mergeDefaults(out.Defaults, overlay.Defaults)
		if overlay.FailFast {
			out.FailFast = true
		}
		if overlay.CacheRemote != nil {
			out.CacheRemote = overlay.CacheRemote
		}
		if overlay.Artifacts != nil {
			out.Artifacts = overlay.Artifacts
		}
		out.Vars = mergeStringMap(out.Vars, overlay.Vars)
		out.Env = mergeStringMap(out.Env, overlay.Env)
		out.Tasks = mergeTaskMap(out.Tasks, overlay.Tasks)
		out.Templates = mergeTaskMap(out.Templates, overlay.Templates)
		out.Plugins = append(out.Plugins, overlay.Plugins...)
		out.LogPlugins = append(out.LogPlugins, overlay.LogPlugins...)
		out.Secrets = append(out.Secrets, overlay.Secrets...)
	}

	return out
}

func mergeStringMap(base, overlay map[string]string) map[string]string {
	if base == nil && overlay == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func mergeTaskMap(base, overlay map[string]*Task) map[string]*Task {
	out := map[string]*Task{}
	for key, task := range base {
		out[key] = cloneTask(task)
	}
	for key, task := range overlay {
		out[key] = cloneTask(task)
	}
	return out
}

func cloneTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	out := *task
	out.Deps = append([]string(nil), task.Deps...)
	out.Needs = append([]string(nil), task.Needs...)
	out.DependsOn = append([]ConditionalDep(nil), task.DependsOn...)
	out.Run = append([]string(nil), task.Run...)
	out.Pre = append([]string(nil), task.Pre...)
	out.Post = append([]string(nil), task.Post...)
	out.Aliases = append([]string(nil), task.Aliases...)
	out.Tags = append([]string(nil), task.Tags...)
	out.Secrets = append([]string(nil), task.Secrets...)
	out.Inputs = append([]string(nil), task.Inputs...)
	out.Outputs = append([]string(nil), task.Outputs...)
	out.OutputPaths = append([]string(nil), task.OutputPaths...)
	if task.Output != nil {
		out.Output = map[string]string{}
		for k, v := range task.Output {
			out.Output[k] = v
		}
	}
	if task.Capture != nil {
		capture := *task.Capture
		out.Capture = &capture
	}
	out.Watch = append([]string(nil), task.Watch...)
	out.Artifacts = append([]string(nil), task.Artifacts...)
	out.RetryOnExitCodes = append([]int(nil), task.RetryOnExitCodes...)
	out.RetryOnRegex = append([]string(nil), task.RetryOnRegex...)
	out.RetryOnSignal = append([]string(nil), task.RetryOnSignal...)
	out.Require = append([]string(nil), task.Require...)
	out.Fanout = task.Fanout
	out.Isolate = task.Isolate
	out.ContinueOnError = task.ContinueOnError
	out.AllowFailure = task.AllowFailure
	out.Silent = task.Silent
	out.IfMissing = task.IfMissing
	out.MaxRetries = task.MaxRetries
	out.Jitter = task.Jitter
	out.Priority = task.Priority
	out.Group = task.Group
	out.RunDir = task.RunDir
	out.Seed = task.Seed
	if task.Env != nil {
		out.Env = map[string]string{}
		for k, v := range task.Env {
			out.Env[k] = v
		}
	}
	if task.Vars != nil {
		out.Vars = map[string]string{}
		for k, v := range task.Vars {
			out.Vars[k] = v
		}
	}
	if task.SeedEnv != nil {
		out.SeedEnv = map[string]string{}
		for k, v := range task.SeedEnv {
			out.SeedEnv[k] = v
		}
	}
	if task.Exports != nil {
		out.Exports = map[string]string{}
		for k, v := range task.Exports {
			out.Exports[k] = v
		}
	}
	if task.OnlyOn != nil {
		only := *task.OnlyOn
		only.Branches = append([]string(nil), task.OnlyOn.Branches...)
		only.Tags = append([]string(nil), task.OnlyOn.Tags...)
		out.OnlyOn = &only
	}
	if task.With != nil {
		out.With = map[string]string{}
		for k, v := range task.With {
			out.With[k] = v
		}
	}
	if task.Matrix != nil {
		out.Matrix = map[string][]string{}
		for k, v := range task.Matrix {
			out.Matrix[k] = append([]string(nil), v...)
		}
	}
	if task.Sweep != nil {
		out.Sweep = cloneSweepSpec(task.Sweep)
	}
	if task.Limits != nil {
		limits := *task.Limits
		out.Limits = &limits
	}
	if task.Resources != nil {
		res := *task.Resources
		out.Resources = &res
	}
	if task.Remote != nil {
		remote := *task.Remote
		remote.Hosts = append([]string(nil), task.Remote.Hosts...)
		if task.Remote.Scheduler != nil {
			scheduler := *task.Remote.Scheduler
			scheduler.Args = append([]string(nil), task.Remote.Scheduler.Args...)
			remote.Scheduler = &scheduler
		}
		out.Remote = &remote
	}
	if task.Scheduler != nil {
		scheduler := *task.Scheduler
		scheduler.Args = append([]string(nil), task.Scheduler.Args...)
		out.Scheduler = &scheduler
	}
	if len(task.Datasets) > 0 {
		out.Datasets = append([]DatasetRef(nil), task.Datasets...)
	}
	if len(task.DatasetOutputs) > 0 {
		out.DatasetOutputs = append([]DatasetOutput(nil), task.DatasetOutputs...)
	}
	if task.Split != nil {
		out.Split = cloneSplitSpec(task.Split)
	}
	if task.Validate != nil {
		out.Validate = cloneValidateSpec(task.Validate)
	}
	if task.Stats != nil {
		out.Stats = cloneStatsSpec(task.Stats)
	}
	if task.Metrics != nil {
		out.Metrics = cloneMetricsSpec(task.Metrics)
	}
	if task.Canary != nil {
		out.Canary = cloneCanarySpec(task.Canary)
	}
	if task.Benchmark != nil {
		out.Benchmark = cloneBenchmarkSpec(task.Benchmark)
	}
	if task.Experiment != nil {
		out.Experiment = cloneExperimentSpec(task.Experiment)
	}
	if task.Snapshot != nil {
		out.Snapshot = cloneSnapshotSpec(task.Snapshot)
	}
	if task.Sign != nil {
		out.Sign = cloneSignSpec(task.Sign)
	}
	if task.SBOM != nil {
		out.SBOM = cloneSBOMSpec(task.SBOM)
	}
	if task.Checkpoint != nil {
		out.Checkpoint = cloneCheckpointSpec(task.Checkpoint)
	}
	if task.ModelCard != nil {
		out.ModelCard = cloneModelCardSpec(task.ModelCard)
	}
	if task.Notebook != nil {
		out.Notebook = cloneNotebookSpec(task.Notebook)
	}
	if task.Export != nil {
		out.Export = cloneExportSpec(task.Export)
	}
	if task.Offline != nil {
		out.Offline = cloneOfflineSpec(task.Offline)
	}
	if task.Docker != nil {
		out.Docker = cloneDocker(task.Docker)
	}
	return &out
}

func cloneDocker(spec *DockerSpec) *DockerSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	if spec.Build != nil {
		build := *spec.Build
		if spec.Build.Args != nil {
			build.Args = map[string]string{}
			for k, v := range spec.Build.Args {
				build.Args[k] = v
			}
		}
		out.Build = &build
	}
	if spec.Push != nil {
		push := *spec.Push
		out.Push = &push
	}
	if spec.Pull != nil {
		pull := *spec.Pull
		out.Pull = &pull
	}
	return &out
}

func autoIncludeDir(basePath string) []string {
	dir := filepath.Join(filepath.Dir(basePath), ".vbuild.d")
	return listYamlFiles(dir)
}

func ioReadAll(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func listYamlFiles(root string) []string {
	entries := []string{}
	if _, err := os.Stat(root); err != nil {
		return entries
	}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			entries = append(entries, path)
		}
		return nil
	})
	sort.Strings(entries)
	return entries
}

func expandIncludes(basePath string, includes []string) ([]string, error) {
	out := []string{}
	for _, inc := range includes {
		inc = strings.TrimSpace(inc)
		if inc == "" {
			continue
		}
		if isURL(inc) {
			out = append(out, inc)
			continue
		}
		resolved, err := resolveInclude(basePath, inc)
		if err != nil {
			return nil, err
		}
		if !hasGlob(resolved) {
			out = append(out, resolved)
			continue
		}
		matches, err := doublestar.FilepathGlob(resolved)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("include pattern matched no files: %s", inc)
		}
		sort.Strings(matches)
		out = append(out, matches...)
	}
	return dedupeStrings(out), nil
}

func hasGlob(path string) bool {
	return strings.ContainsAny(path, "*?[") || strings.Contains(path, "**")
}

func configHash(sources map[string][]byte) string {
	keys := sortedKeys(sources)
	h := sha256.New()
	for _, key := range keys {
		_, _ = h.Write([]byte(key))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(sources[key])
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sortedKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mergeDefaults(base, overlay *Defaults) *Defaults {
	if base == nil && overlay == nil {
		return nil
	}
	out := &Defaults{}
	if base != nil {
		*out = *base
	}
	if overlay != nil {
		if overlay.Timeout != "" {
			out.Timeout = overlay.Timeout
		}
		if overlay.Shell != "" {
			out.Shell = overlay.Shell
		}
		if overlay.Workdir != "" {
			out.Workdir = overlay.Workdir
		}
		if overlay.Retries != 0 {
			out.Retries = overlay.Retries
		}
		if overlay.MaxRetries != 0 {
			out.MaxRetries = overlay.MaxRetries
		}
		if overlay.Backoff != "" {
			out.Backoff = overlay.Backoff
		}
		if overlay.Jitter != "" {
			out.Jitter = overlay.Jitter
		}
	}
	return out
}
