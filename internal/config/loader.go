package config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	cfg, err := loadRecursive(path, map[string]bool{})
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

func loadRecursive(path string, visited map[string]bool) (*Config, error) {
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

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	cfg.normalize()

	base := &Config{}
	base.normalize()

	for _, inc := range autoIncludes {
		resolved, err := resolveInclude(path, inc)
		if err != nil {
			return nil, err
		}
		child, err := loadRecursive(resolved, visited)
		if err != nil {
			return nil, err
		}
		base = mergeConfigs(base, child)
	}

	for _, inc := range cfg.Include {
		inc = strings.TrimSpace(inc)
		if inc == "" {
			continue
		}
		resolved, err := resolveInclude(path, inc)
		if err != nil {
			return nil, err
		}
		child, err := loadRecursive(resolved, visited)
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
		out.CacheRemote = base.CacheRemote
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
		if overlay.CacheRemote != nil {
			out.CacheRemote = overlay.CacheRemote
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
	out.Tags = append([]string(nil), task.Tags...)
	out.Secrets = append([]string(nil), task.Secrets...)
	out.Inputs = append([]string(nil), task.Inputs...)
	out.Outputs = append([]string(nil), task.Outputs...)
	out.OutputPaths = append([]string(nil), task.OutputPaths...)
	out.Watch = append([]string(nil), task.Watch...)
	out.Artifacts = append([]string(nil), task.Artifacts...)
	out.RetryOnExitCodes = append([]int(nil), task.RetryOnExitCodes...)
	out.RetryOnRegex = append([]string(nil), task.RetryOnRegex...)
	out.Fanout = task.Fanout
	out.Isolate = task.Isolate
	out.ContinueOnError = task.ContinueOnError
	out.AllowFailure = task.AllowFailure
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
	if task.Exports != nil {
		out.Exports = map[string]string{}
		for k, v := range task.Exports {
			out.Exports[k] = v
		}
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
	if task.Limits != nil {
		limits := *task.Limits
		out.Limits = &limits
	}
	if task.Remote != nil {
		remote := *task.Remote
		out.Remote = &remote
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
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			names = append(names, filepath.Join(dir, name))
		}
	}
	sort.Strings(names)
	return names
}

func ioReadAll(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return body, nil
}
