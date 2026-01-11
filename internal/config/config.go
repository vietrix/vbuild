package config

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Config struct {
	Path         string            `yaml:"-"`
	Hash         string            `yaml:"-"`
	Workflow     string            `yaml:"workflow"`
	Include      []string          `yaml:"include"`
	EnvFile      string            `yaml:"env_file"`
	ArtifactsDir string            `yaml:"artifacts_dir"`
	Timeout      string            `yaml:"timeout"`
	Defaults     *Defaults         `yaml:"defaults"`
	FailFast     bool              `yaml:"fail_fast"`
	CacheRemote  *CacheRemote      `yaml:"cache_remote"`
	Artifacts    *ArtifactsUpload  `yaml:"artifacts_upload"`
	LogPlugins   []Plugin          `yaml:"log_plugins"`
	Secrets      []string          `yaml:"secrets"`
	Vars         map[string]string `yaml:"vars"`
	Env          map[string]string `yaml:"env"`
	Templates    map[string]*Task  `yaml:"templates"`
	Tasks        map[string]*Task  `yaml:"tasks"`
	Plugins      []Plugin          `yaml:"plugins"`
	Aliases      map[string]string `yaml:"-"`
	Sources      []string          `yaml:"-"`
}

type Defaults struct {
	Timeout    string `yaml:"timeout"`
	Shell      string `yaml:"shell"`
	Workdir    string `yaml:"workdir"`
	Retries    int    `yaml:"retries"`
	MaxRetries int    `yaml:"max_retries"`
	Backoff    string `yaml:"backoff"`
	Jitter     string `yaml:"jitter"`
}

type Task struct {
	Desc             string              `yaml:"desc"`
	Aliases          []string            `yaml:"aliases"`
	Deps             []string            `yaml:"deps"`
	Needs            []string            `yaml:"needs"`
	DependsOn        []ConditionalDep    `yaml:"depends_on"`
	Run              []string            `yaml:"run"`
	Pre              []string            `yaml:"pre"`
	Post             []string            `yaml:"post"`
	Env              map[string]string   `yaml:"env"`
	Vars             map[string]string   `yaml:"vars"`
	Parallel         bool                `yaml:"parallel"`
	Fanout           bool                `yaml:"fanout"`
	Workdir          string              `yaml:"workdir"`
	Shell            string              `yaml:"shell"`
	When             string              `yaml:"when"`
	OnlyOn           *OnlyOn             `yaml:"only_on"`
	Tags             []string            `yaml:"tags"`
	Secrets          []string            `yaml:"secrets"`
	Inputs           []string            `yaml:"inputs"`
	Outputs          []string            `yaml:"outputs"`
	OutputPaths      []string            `yaml:"output_paths"`
	Output           map[string]string   `yaml:"output"`
	Capture          *OutputCapture      `yaml:"capture"`
	Exports          map[string]string   `yaml:"exports"`
	Cache            string              `yaml:"cache"`
	Retries          int                 `yaml:"retries"`
	MaxRetries       int                 `yaml:"max_retries"`
	RetryOnExitCodes []int               `yaml:"retry_on_exit_codes"`
	RetryOnRegex     []string            `yaml:"retry_on_regex"`
	RetryOnSignal    []string            `yaml:"retry_on_signal"`
	Backoff          string              `yaml:"backoff"`
	Jitter           string              `yaml:"jitter"`
	Timeout          string              `yaml:"timeout"`
	ContinueOnError  bool                `yaml:"continue_on_error"`
	AllowFailure     bool                `yaml:"allow_failure"`
	Confirm          string              `yaml:"confirm"`
	Isolate          bool                `yaml:"isolate"`
	Silent           bool                `yaml:"silent"`
	IfMissing        bool                `yaml:"if_missing"`
	Require          []string            `yaml:"require"`
	Limits           *ResourceLimits     `yaml:"limits"`
	Remote           *RemoteSpec         `yaml:"remote"`
	Use              string              `yaml:"use"`
	With             map[string]string   `yaml:"with"`
	Matrix           map[string][]string `yaml:"matrix"`
	Watch            []string            `yaml:"watch"`
	Artifacts        []string            `yaml:"artifacts"`
	Docker           *DockerSpec         `yaml:"docker"`
}

type ConditionalDep struct {
	Task string `yaml:"task"`
	When string `yaml:"when"`
}

type OnlyOn struct {
	Branches []string `yaml:"branches"`
	Tags     []string `yaml:"tags"`
}

type OutputCapture struct {
	Stdout   string `yaml:"stdout"`
	Stderr   string `yaml:"stderr"`
	Combined string `yaml:"combined"`
	Append   bool   `yaml:"append"`
}

type ResourceLimits struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type RemoteSpec struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Port     int    `yaml:"port"`
	Identity string `yaml:"identity"`
	Workdir  string `yaml:"workdir"`
}

type CacheRemote struct {
	Provider     string `yaml:"provider"`
	Bucket       string `yaml:"bucket"`
	Prefix       string `yaml:"prefix"`
	Region       string `yaml:"region"`
	Endpoint     string `yaml:"endpoint"`
	AccessKey    string `yaml:"access_key"`
	SecretKey    string `yaml:"secret_key"`
	SessionToken string `yaml:"session_token"`
	PathStyle    bool   `yaml:"path_style"`
}

type ArtifactsUpload struct {
	Provider     string `yaml:"provider"`
	Repo         string `yaml:"repo"`
	Tag          string `yaml:"tag"`
	Token        string `yaml:"token"`
	Bucket       string `yaml:"bucket"`
	Prefix       string `yaml:"prefix"`
	Region       string `yaml:"region"`
	Endpoint     string `yaml:"endpoint"`
	AccessKey    string `yaml:"access_key"`
	SecretKey    string `yaml:"secret_key"`
	SessionToken string `yaml:"session_token"`
	PathStyle    bool   `yaml:"path_style"`
}

type DockerSpec struct {
	Build *DockerBuild `yaml:"build"`
	Push  *DockerPush  `yaml:"push"`
	Pull  *DockerPull  `yaml:"pull"`
}

type DockerBuild struct {
	Context    string            `yaml:"context"`
	Dockerfile string            `yaml:"dockerfile"`
	Tag        string            `yaml:"tag"`
	Platform   string            `yaml:"platform"`
	Target     string            `yaml:"target"`
	Args       map[string]string `yaml:"args"`
}

type DockerPush struct {
	Tag string `yaml:"tag"`
}

type DockerPull struct {
	Tag string `yaml:"tag"`
}

type Plugin struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	var builder strings.Builder
	builder.WriteString("config validation failed:")
	for _, issue := range e.Issues {
		builder.WriteString("\n- ")
		builder.WriteString(issue)
	}
	return builder.String()
}

func (c *Config) normalize() {
	if c.Tasks == nil {
		c.Tasks = map[string]*Task{}
	}
	if c.Templates == nil {
		c.Templates = map[string]*Task{}
	}
	if c.Env == nil {
		c.Env = map[string]string{}
	}
	if c.Vars == nil {
		c.Vars = map[string]string{}
	}
	if c.Plugins == nil {
		c.Plugins = []Plugin{}
	}
	if c.LogPlugins == nil {
		c.LogPlugins = []Plugin{}
	}
	if c.Defaults != nil {
		if c.Defaults.Retries < 0 {
			c.Defaults.Retries = 0
		}
		if c.Defaults.MaxRetries < 0 {
			c.Defaults.MaxRetries = 0
		}
	}
	c.Secrets = dedupeStrings(c.Secrets)
	c.normalizeTasks()
	c.buildAliases()
}

func (c *Config) normalizeTasks() {
	for _, task := range c.Tasks {
		if task == nil {
			continue
		}
		if task.Env == nil {
			task.Env = map[string]string{}
		}
		if task.Vars == nil {
			task.Vars = map[string]string{}
		}
		if task.Output == nil {
			task.Output = map[string]string{}
		}
		if task.Exports == nil {
			task.Exports = map[string]string{}
		}
		if task.With != nil {
			for key, value := range task.With {
				task.Vars[key] = value
			}
		}
		if len(task.Needs) > 0 {
			task.Deps = append(task.Deps, task.Needs...)
			task.Needs = nil
		}
		if len(task.OutputPaths) == 0 && len(task.Outputs) > 0 {
			task.OutputPaths = append(task.OutputPaths, task.Outputs...)
		}
		if task.OnlyOn != nil {
			task.OnlyOn.Branches = dedupeStrings(task.OnlyOn.Branches)
			task.OnlyOn.Tags = dedupeStrings(task.OnlyOn.Tags)
		}
		task.Aliases = dedupeStrings(task.Aliases)
		task.Deps = dedupeStrings(task.Deps)
		task.Tags = dedupeStrings(task.Tags)
		task.Secrets = dedupeStrings(task.Secrets)
		task.Inputs = dedupeStrings(task.Inputs)
		task.Outputs = dedupeStrings(task.Outputs)
		task.OutputPaths = dedupeStrings(task.OutputPaths)
		task.RetryOnRegex = dedupeStrings(task.RetryOnRegex)
		task.RetryOnSignal = dedupeStrings(task.RetryOnSignal)
		task.Require = dedupeStrings(task.Require)
	}
}

func (c *Config) validate() error {
	issues := []string{}

	validateKeyMap("vars", c.Vars, &issues)
	validateKeyMap("env", c.Env, &issues)
	if c.Defaults != nil {
		if strings.TrimSpace(c.Defaults.Workdir) == "" && c.Defaults.Workdir != "" {
			issues = append(issues, "defaults.workdir must not be empty")
		}
		if strings.TrimSpace(c.Defaults.Shell) == "" && c.Defaults.Shell != "" {
			issues = append(issues, "defaults.shell must not be empty")
		}
		if c.Defaults.Timeout != "" {
			if _, err := time.ParseDuration(c.Defaults.Timeout); err != nil {
				issues = append(issues, fmt.Sprintf("defaults.timeout invalid duration: %s", err))
			}
		}
		if c.Defaults.Backoff != "" {
			if _, err := time.ParseDuration(c.Defaults.Backoff); err != nil {
				issues = append(issues, fmt.Sprintf("defaults.backoff invalid duration: %s", err))
			}
		}
		if c.Defaults.Jitter != "" {
			if _, err := time.ParseDuration(c.Defaults.Jitter); err != nil {
				issues = append(issues, fmt.Sprintf("defaults.jitter invalid duration: %s", err))
			}
		}
		if c.Defaults.Retries < 0 {
			issues = append(issues, "defaults.retries must be >= 0")
		}
		if c.Defaults.MaxRetries < 0 {
			issues = append(issues, "defaults.max_retries must be >= 0")
		}
		if c.Defaults.MaxRetries > 0 && c.Defaults.Retries > c.Defaults.MaxRetries {
			issues = append(issues, "defaults.retries must be <= defaults.max_retries")
		}
	}
	if c.ArtifactsDir != "" && strings.TrimSpace(c.ArtifactsDir) == "" {
		issues = append(issues, "artifacts_dir must not be empty")
	}
	if c.EnvFile != "" && strings.TrimSpace(c.EnvFile) == "" {
		issues = append(issues, "env_file must not be empty")
	}
	if c.Timeout != "" {
		if _, err := time.ParseDuration(c.Timeout); err != nil {
			issues = append(issues, fmt.Sprintf("timeout invalid duration: %s", err))
		}
	}
	for i, plugin := range c.Plugins {
		if strings.TrimSpace(plugin.Command) == "" {
			issues = append(issues, fmt.Sprintf("plugins[%d].command must not be empty", i))
		}
	}
	for i, plugin := range c.LogPlugins {
		if strings.TrimSpace(plugin.Command) == "" {
			issues = append(issues, fmt.Sprintf("log_plugins[%d].command must not be empty", i))
		}
	}

	for i, item := range c.Include {
		if strings.TrimSpace(item) == "" {
			issues = append(issues, fmt.Sprintf("include[%d] must not be empty", i))
		}
	}
	for i, secret := range c.Secrets {
		if strings.TrimSpace(secret) == "" {
			issues = append(issues, fmt.Sprintf("secrets[%d] must not be empty", i))
		}
	}
	if c.CacheRemote != nil {
		provider := strings.ToLower(strings.TrimSpace(c.CacheRemote.Provider))
		if provider == "" {
			issues = append(issues, "cache_remote.provider must not be empty")
		} else if provider != "s3" && provider != "gcs" && provider != "minio" {
			issues = append(issues, fmt.Sprintf("cache_remote.provider must be s3, gcs, or minio (got %q)", c.CacheRemote.Provider))
		}
		if strings.TrimSpace(c.CacheRemote.Bucket) == "" {
			issues = append(issues, "cache_remote.bucket must not be empty")
		}
	}
	if c.Artifacts != nil {
		provider := strings.ToLower(strings.TrimSpace(c.Artifacts.Provider))
		if provider == "" {
			issues = append(issues, "artifacts_upload.provider must not be empty")
		} else if provider != "github" && provider != "s3" {
			issues = append(issues, fmt.Sprintf("artifacts_upload.provider must be github or s3 (got %q)", c.Artifacts.Provider))
		}
		switch provider {
		case "github":
			if strings.TrimSpace(c.Artifacts.Repo) == "" {
				issues = append(issues, "artifacts_upload.repo must not be empty")
			}
		case "s3":
			if strings.TrimSpace(c.Artifacts.Bucket) == "" {
				issues = append(issues, "artifacts_upload.bucket must not be empty")
			}
		}
	}

	for name, tmpl := range c.Templates {
		if strings.TrimSpace(name) == "" {
			issues = append(issues, "templates key must not be empty")
			continue
		}
		if tmpl == nil {
			issues = append(issues, fmt.Sprintf("templates.%s must be an object", name))
			continue
		}
		validateTask(fmt.Sprintf("templates.%s", name), tmpl, c, &issues, false)
	}

	if len(c.Tasks) == 0 {
		issues = append(issues, "tasks must not be empty")
	}

	aliases := map[string]string{}

	taskNames := make([]string, 0, len(c.Tasks))
	for name := range c.Tasks {
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)

	for _, name := range taskNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			issues = append(issues, "task name must not be empty")
			continue
		}
		task := c.Tasks[name]
		if task == nil {
			issues = append(issues, fmt.Sprintf("tasks.%s must be an object", name))
			continue
		}

		if len(task.Deps) == 0 && len(task.Run) == 0 && task.Docker == nil {
			issues = append(issues, fmt.Sprintf("tasks.%s must define deps or run", name))
		}

		validateTask(fmt.Sprintf("tasks.%s", name), task, c, &issues, true)

		for i, alias := range task.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				issues = append(issues, fmt.Sprintf("tasks.%s.aliases[%d] must not be empty", name, i))
				continue
			}
			if _, ok := c.Tasks[alias]; ok {
				issues = append(issues, fmt.Sprintf("tasks.%s.aliases[%d] conflicts with task name %q", name, i, alias))
				continue
			}
			if existing, ok := aliases[alias]; ok && existing != name {
				issues = append(issues, fmt.Sprintf("tasks.%s.aliases[%d] duplicates alias for %q", name, i, existing))
				continue
			}
			aliases[alias] = name
		}
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

func validateKeyMap(path string, values map[string]string, issues *[]string) {
	if values == nil {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			*issues = append(*issues, fmt.Sprintf("%s key must not be empty", path))
		}
	}
}

func validateTask(path string, task *Task, cfg *Config, issues *[]string, checkDeps bool) {
	for i, alias := range task.Aliases {
		if strings.TrimSpace(alias) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.aliases[%d] must not be empty", path, i))
		}
	}
	for i, dep := range task.Deps {
		depName := strings.TrimSpace(dep)
		if depName == "" {
			*issues = append(*issues, fmt.Sprintf("%s.deps[%d] must not be empty", path, i))
			continue
		}
		if checkDeps {
			if _, ok := cfg.Tasks[depName]; !ok {
				*issues = append(*issues, fmt.Sprintf("%s.deps[%d] refers to unknown task %q", path, i, depName))
			}
		}
	}

	for i, cmd := range task.Run {
		if strings.TrimSpace(cmd) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.run[%d] must not be empty", path, i))
		}
	}
	for i, cmd := range task.Pre {
		if strings.TrimSpace(cmd) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.pre[%d] must not be empty", path, i))
		}
	}
	for i, cmd := range task.Post {
		if strings.TrimSpace(cmd) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.post[%d] must not be empty", path, i))
		}
	}
	for i, tag := range task.Tags {
		if strings.TrimSpace(tag) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.tags[%d] must not be empty", path, i))
		}
	}
	for i, secret := range task.Secrets {
		if strings.TrimSpace(secret) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.secrets[%d] must not be empty", path, i))
		}
	}
	for i, input := range task.Inputs {
		if strings.TrimSpace(input) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.inputs[%d] must not be empty", path, i))
		}
	}
	for i, output := range task.Outputs {
		if strings.TrimSpace(output) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.outputs[%d] must not be empty", path, i))
		}
	}
	for i, output := range task.OutputPaths {
		if strings.TrimSpace(output) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.output_paths[%d] must not be empty", path, i))
		}
	}
	for key, value := range task.Output {
		if strings.TrimSpace(key) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.output key must not be empty", path))
			continue
		}
		if strings.TrimSpace(value) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.output.%s must not be empty", path, key))
		}
	}
	if task.Capture != nil {
		if strings.TrimSpace(task.Capture.Stdout) == "" && strings.TrimSpace(task.Capture.Stderr) == "" && strings.TrimSpace(task.Capture.Combined) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.capture must set stdout, stderr, or combined", path))
		}
	}
	for key, value := range task.Exports {
		if strings.TrimSpace(key) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.exports key must not be empty", path))
		}
		if strings.TrimSpace(value) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.exports.%s must not be empty", path, key))
		}
	}
	for i, watch := range task.Watch {
		if strings.TrimSpace(watch) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.watch[%d] must not be empty", path, i))
		}
	}
	if task.OnlyOn != nil {
		for i, pattern := range task.OnlyOn.Branches {
			if strings.TrimSpace(pattern) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.only_on.branches[%d] must not be empty", path, i))
			}
		}
		for i, pattern := range task.OnlyOn.Tags {
			if strings.TrimSpace(pattern) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.only_on.tags[%d] must not be empty", path, i))
			}
		}
	}
	if task.Use != "" {
		if _, ok := cfg.Templates[task.Use]; !ok {
			*issues = append(*issues, fmt.Sprintf("%s.use refers to unknown template %q", path, task.Use))
		}
	}
	for i, dep := range task.DependsOn {
		if strings.TrimSpace(dep.Task) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.depends_on[%d].task must not be empty", path, i))
			continue
		}
		if checkDeps {
			if _, ok := cfg.Tasks[dep.Task]; !ok {
				*issues = append(*issues, fmt.Sprintf("%s.depends_on[%d] refers to unknown task %q", path, i, dep.Task))
			}
		}
	}
	if task.Retries < 0 {
		*issues = append(*issues, fmt.Sprintf("%s.retries must be >= 0", path))
	}
	if task.MaxRetries < 0 {
		*issues = append(*issues, fmt.Sprintf("%s.max_retries must be >= 0", path))
	}
	if task.MaxRetries > 0 && task.Retries > task.MaxRetries {
		*issues = append(*issues, fmt.Sprintf("%s.retries must be <= %s.max_retries", path, path))
	}
	if task.Backoff != "" {
		if _, err := time.ParseDuration(task.Backoff); err != nil {
			*issues = append(*issues, fmt.Sprintf("%s.backoff invalid duration: %s", path, err))
		}
	}
	if task.Jitter != "" {
		if _, err := time.ParseDuration(task.Jitter); err != nil {
			*issues = append(*issues, fmt.Sprintf("%s.jitter invalid duration: %s", path, err))
		}
	}
	if task.Timeout != "" {
		if _, err := time.ParseDuration(task.Timeout); err != nil {
			*issues = append(*issues, fmt.Sprintf("%s.timeout invalid duration: %s", path, err))
		}
	}
	if task.Limits != nil {
		if task.Limits.CPU != "" {
			if _, err := time.ParseDuration(task.Limits.CPU); err != nil {
				*issues = append(*issues, fmt.Sprintf("%s.limits.cpu invalid duration: %s", path, err))
			}
		}
		if task.Limits.Memory != "" && strings.TrimSpace(task.Limits.Memory) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.limits.memory must not be empty", path))
		}
	}
	if task.Cache != "" && task.Cache != "mtime" && task.Cache != "sha256" {
		*issues = append(*issues, fmt.Sprintf("%s.cache must be mtime or sha256", path))
	}
	for i, code := range task.RetryOnExitCodes {
		if code < 0 {
			*issues = append(*issues, fmt.Sprintf("%s.retry_on_exit_codes[%d] must be >= 0", path, i))
		}
	}
	for i, pattern := range task.RetryOnRegex {
		if strings.TrimSpace(pattern) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.retry_on_regex[%d] must not be empty", path, i))
		}
	}
	for i, pattern := range task.RetryOnSignal {
		if strings.TrimSpace(pattern) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.retry_on_signal[%d] must not be empty", path, i))
		}
	}
	validateKeyMap(fmt.Sprintf("%s.env", path), task.Env, issues)
	validateKeyMap(fmt.Sprintf("%s.vars", path), task.Vars, issues)

	for key, values := range task.Matrix {
		if strings.TrimSpace(key) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.matrix key must not be empty", path))
			continue
		}
		if len(values) == 0 {
			*issues = append(*issues, fmt.Sprintf("%s.matrix.%s must not be empty", path, key))
		}
	}

	if task.Docker != nil {
		if task.Docker.Build == nil && task.Docker.Push == nil && task.Docker.Pull == nil {
			*issues = append(*issues, fmt.Sprintf("%s.docker must define build, push, or pull", path))
		}
	}
	if task.Remote != nil {
		if strings.TrimSpace(task.Remote.Host) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.remote.host must not be empty", path))
		}
	}
	if task.IfMissing {
		if len(task.Outputs) == 0 && len(task.OutputPaths) == 0 {
			*issues = append(*issues, fmt.Sprintf("%s.if_missing requires outputs or output_paths", path))
		}
	}
	for i, req := range task.Require {
		if strings.TrimSpace(req) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.require[%d] must not be empty", path, i))
		}
	}
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func (c *Config) buildAliases() {
	c.Aliases = map[string]string{}
	for name, task := range c.Tasks {
		if task == nil {
			continue
		}
		for _, alias := range task.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			c.Aliases[alias] = name
		}
	}
}
