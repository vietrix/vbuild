package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Path         string              `yaml:"-"`
	Hash         string              `yaml:"-"`
	Workflow     string              `yaml:"workflow"`
	Include      []string            `yaml:"include"`
	EnvFile      string              `yaml:"env_file"`
	ArtifactsDir string              `yaml:"artifacts_dir"`
	Timeout      string              `yaml:"timeout"`
	Seed         int64               `yaml:"seed"`
	SeedEnv      map[string]string   `yaml:"seed_env"`
	Offline      *OfflineSpec        `yaml:"offline"`
	Resources    *ResourcePool       `yaml:"resources"`
	Datasets     map[string]*Dataset `yaml:"datasets"`
	Experiments  *ExperimentDefaults `yaml:"experiments"`
	Registry     *RegistrySpec       `yaml:"registry"`
	Snapshot     *SnapshotSpec       `yaml:"snapshot"`
	Defaults     *Defaults           `yaml:"defaults"`
	FailFast     bool                `yaml:"fail_fast"`
	CacheRemote  *CacheRemote        `yaml:"cache_remote"`
	Artifacts    *ArtifactsUpload    `yaml:"artifacts_upload"`
	LogPlugins   []Plugin            `yaml:"log_plugins"`
	Secrets      []string            `yaml:"secrets"`
	Vars         map[string]string   `yaml:"vars"`
	Env          map[string]string   `yaml:"env"`
	Templates    map[string]*Task    `yaml:"templates"`
	Tasks        map[string]*Task    `yaml:"tasks"`
	Plugins      []Plugin            `yaml:"plugins"`
	Aliases      map[string]string   `yaml:"-"`
	Sources      []string            `yaml:"-"`
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
	PassArgs         bool                `yaml:"pass_args"`
	Deps             []string            `yaml:"deps"`
	Needs            []string            `yaml:"needs"`
	DependsOn        []ConditionalDep    `yaml:"depends_on"`
	Script           string              `yaml:"script"`
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
	Resources        *ResourceRequest    `yaml:"resources"`
	Priority         int                 `yaml:"priority"`
	Group            string              `yaml:"group"`
	RunDir           string              `yaml:"run_dir"`
	Seed             int64               `yaml:"seed"`
	SeedEnv          map[string]string   `yaml:"seed_env"`
	Remote           *RemoteSpec         `yaml:"remote"`
	Scheduler        *SchedulerSpec      `yaml:"scheduler"`
	Use              string              `yaml:"use"`
	With             map[string]string   `yaml:"with"`
	Matrix           map[string][]string `yaml:"matrix"`
	Sweep            *SweepSpec          `yaml:"sweep"`
	Watch            []string            `yaml:"watch"`
	Artifacts        []string            `yaml:"artifacts"`
	Docker           *DockerSpec         `yaml:"docker"`
	Datasets         []DatasetRef        `yaml:"datasets"`
	DatasetOutputs   []DatasetOutput     `yaml:"dataset_outputs"`
	Split            *SplitSpec          `yaml:"split"`
	Validate         *ValidateSpec       `yaml:"validate"`
	Stats            *StatsSpec          `yaml:"stats"`
	Metrics          *MetricsSpec        `yaml:"metrics"`
	Canary           *CanarySpec         `yaml:"canary"`
	Benchmark        *BenchmarkSpec      `yaml:"benchmark"`
	Experiment       *ExperimentSpec     `yaml:"experiment"`
	Snapshot         *SnapshotSpec       `yaml:"snapshot"`
	Sign             *SignSpec           `yaml:"sign"`
	SBOM             *SBOMSpec           `yaml:"sbom"`
	Checkpoint       *CheckpointSpec     `yaml:"checkpoint"`
	ModelCard        *ModelCardSpec      `yaml:"model_card"`
	Notebook         *NotebookSpec       `yaml:"notebook"`
	Export           *ExportSpec         `yaml:"export"`
	Offline          *OfflineSpec        `yaml:"offline"`
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

type ResourcePool struct {
	CPU        int            `yaml:"cpu"`
	Memory     string         `yaml:"memory"`
	GPUs       int            `yaml:"gpus"`
	GPUDevices []string       `yaml:"gpu_devices"`
	Groups     map[string]int `yaml:"groups"`
}

type ResourceRequest struct {
	CPU    int    `yaml:"cpu"`
	Memory string `yaml:"memory"`
	GPU    int    `yaml:"gpu"`
	Group  string `yaml:"group"`
}

type Dataset struct {
	Path     string   `yaml:"path"`
	Files    []string `yaml:"files"`
	Desc     string   `yaml:"desc"`
	Tags     []string `yaml:"tags"`
	Version  string   `yaml:"version"`
	Manifest string   `yaml:"manifest"`
	Format   string   `yaml:"format"`
}

type DatasetRef struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	Optional bool   `yaml:"optional"`
}

type DatasetOutput struct {
	Name    string   `yaml:"name"`
	Path    string   `yaml:"path"`
	Files   []string `yaml:"files"`
	Tags    []string `yaml:"tags"`
	Version string   `yaml:"version"`
}

type ExperimentDefaults struct {
	Dir      string            `yaml:"dir"`
	Enabled  bool              `yaml:"enabled"`
	Tags     []string          `yaml:"tags"`
	Metadata map[string]string `yaml:"metadata"`
}

type ExperimentSpec struct {
	Name     string            `yaml:"name"`
	Tags     []string          `yaml:"tags"`
	Record   *bool             `yaml:"record"`
	Metadata map[string]string `yaml:"metadata"`
}

type MetricsSpec struct {
	Regex  []string `yaml:"regex"`
	File   string   `yaml:"file"`
	Format string   `yaml:"format"`
	Prefix string   `yaml:"prefix"`
}

type CanarySpec struct {
	Baseline     string                `yaml:"baseline"`
	Rules        map[string]CanaryRule `yaml:"rules"`
	AllowMissing bool                  `yaml:"allow_missing"`
}

type CanaryRule struct {
	Min         *float64 `yaml:"min"`
	Max         *float64 `yaml:"max"`
	MaxDelta    *float64 `yaml:"max_delta"`
	MaxDeltaPct *float64 `yaml:"max_delta_pct"`
}

type BenchmarkSpec struct {
	Iterations int    `yaml:"iterations"`
	Warmup     int    `yaml:"warmup"`
	Output     string `yaml:"output"`
}

type SweepSpec struct {
	Seed    int64               `yaml:"seed"`
	Samples int                 `yaml:"samples"`
	Grid    map[string][]string `yaml:"grid"`
	Sample  map[string][]string `yaml:"sample"`
}

type SplitSpec struct {
	Input   string  `yaml:"input"`
	Output  string  `yaml:"output"`
	Seed    int64   `yaml:"seed"`
	Train   float64 `yaml:"train"`
	Val     float64 `yaml:"val"`
	Test    float64 `yaml:"test"`
	Shuffle bool    `yaml:"shuffle"`
	Format  string  `yaml:"format"`
}

type ValidateSpec struct {
	Paths       []string `yaml:"paths"`
	MinFiles    int      `yaml:"min_files"`
	MaxFiles    int      `yaml:"max_files"`
	MinSize     string   `yaml:"min_size"`
	MaxSize     string   `yaml:"max_size"`
	Extensions  []string `yaml:"extensions"`
	SampleRegex string   `yaml:"sample_regex"`
}

type StatsSpec struct {
	Paths  []string `yaml:"paths"`
	Output string   `yaml:"output"`
	Lines  bool     `yaml:"lines"`
	Hash   bool     `yaml:"hash"`
}

type SnapshotSpec struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
	Env     bool   `yaml:"env"`
	Vars    bool   `yaml:"vars"`
	Git     bool   `yaml:"git"`
	System  bool   `yaml:"system"`
}

type SignSpec struct {
	Output string `yaml:"output"`
}

type SBOMSpec struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"`
}

type CheckpointSpec struct {
	Paths []string `yaml:"paths"`
	Var   string   `yaml:"var"`
}

type ModelCardSpec struct {
	Path     string            `yaml:"path"`
	Template string            `yaml:"template"`
	Metadata map[string]string `yaml:"metadata"`
}

type NotebookSpec struct {
	Path       string            `yaml:"path"`
	Output     string            `yaml:"output"`
	Kernel     string            `yaml:"kernel"`
	Parameters map[string]string `yaml:"parameters"`
}

type ExportSpec struct {
	Path    string   `yaml:"path"`
	Format  string   `yaml:"format"`
	Include []string `yaml:"include"`
}

type OfflineSpec struct {
	Enabled bool              `yaml:"enabled"`
	Env     map[string]string `yaml:"env"`
}

type RegistrySpec struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

type RemoteSpec struct {
	Host      string         `yaml:"host"`
	Hosts     []string       `yaml:"hosts"`
	User      string         `yaml:"user"`
	Port      int            `yaml:"port"`
	Identity  string         `yaml:"identity"`
	Workdir   string         `yaml:"workdir"`
	Scheduler *SchedulerSpec `yaml:"scheduler"`
}

type SchedulerSpec struct {
	Type    string   `yaml:"type"`
	Queue   string   `yaml:"queue"`
	Account string   `yaml:"account"`
	Time    string   `yaml:"time"`
	Nodes   int      `yaml:"nodes"`
	GPUs    int      `yaml:"gpus"`
	CPUs    int      `yaml:"cpus"`
	Memory  string   `yaml:"memory"`
	Args    []string `yaml:"args"`
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
	if c.SeedEnv == nil {
		c.SeedEnv = map[string]string{}
	}
	if c.Datasets == nil {
		c.Datasets = map[string]*Dataset{}
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
	if c.Experiments != nil {
		c.Experiments.Tags = dedupeStrings(c.Experiments.Tags)
		if c.Experiments.Metadata == nil {
			c.Experiments.Metadata = map[string]string{}
		}
	}
	if c.Offline != nil && c.Offline.Env == nil {
		c.Offline.Env = map[string]string{}
	}
	if c.Resources != nil {
		c.Resources.GPUDevices = dedupeStrings(c.Resources.GPUDevices)
		if c.Resources.Groups == nil {
			c.Resources.Groups = map[string]int{}
		}
	}
	if c.Snapshot != nil {
		if c.Snapshot.Path != "" {
			c.Snapshot.Path = strings.TrimSpace(c.Snapshot.Path)
		}
	}
	if c.Registry != nil && c.Registry.Type == "" {
		c.Registry.Type = "local"
	}
	c.Secrets = dedupeStrings(c.Secrets)
	c.normalizeDatasets()
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
		if task.SeedEnv == nil {
			task.SeedEnv = map[string]string{}
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
		task.Script = strings.TrimSpace(task.Script)
		if task.OnlyOn != nil {
			task.OnlyOn.Branches = dedupeStrings(task.OnlyOn.Branches)
			task.OnlyOn.Tags = dedupeStrings(task.OnlyOn.Tags)
		}
		if task.Resources != nil && task.Resources.Group == "" {
			task.Resources.Group = strings.TrimSpace(task.Group)
		}
		if task.Experiment != nil {
			task.Experiment.Tags = dedupeStrings(task.Experiment.Tags)
			if task.Experiment.Metadata == nil {
				task.Experiment.Metadata = map[string]string{}
			}
		}
		if task.Offline != nil && task.Offline.Env == nil {
			task.Offline.Env = map[string]string{}
		}
		if task.Sweep != nil {
			for key := range task.Sweep.Grid {
				task.Sweep.Grid[key] = dedupeStrings(task.Sweep.Grid[key])
			}
			for key := range task.Sweep.Sample {
				task.Sweep.Sample[key] = dedupeStrings(task.Sweep.Sample[key])
			}
		}
		if task.Split != nil {
			task.Split.Input = strings.TrimSpace(task.Split.Input)
			task.Split.Output = strings.TrimSpace(task.Split.Output)
		}
		if task.Validate != nil {
			task.Validate.Paths = dedupeStrings(task.Validate.Paths)
			task.Validate.Extensions = dedupeStrings(task.Validate.Extensions)
		}
		if task.Stats != nil {
			task.Stats.Paths = dedupeStrings(task.Stats.Paths)
		}
		if task.Metrics != nil {
			task.Metrics.Regex = dedupeStrings(task.Metrics.Regex)
		}
		if task.Datasets != nil {
			task.Datasets = dedupeDatasetRefs(task.Datasets)
		}
		if task.DatasetOutputs != nil {
			task.DatasetOutputs = normalizeDatasetOutputs(task.DatasetOutputs)
		}
		if task.Notebook != nil && task.Notebook.Parameters == nil {
			task.Notebook.Parameters = map[string]string{}
		}
		if task.ModelCard != nil && task.ModelCard.Metadata == nil {
			task.ModelCard.Metadata = map[string]string{}
		}
		if task.Export != nil {
			task.Export.Include = dedupeStrings(task.Export.Include)
		}
		if task.Remote != nil {
			task.Remote.Host = strings.TrimSpace(task.Remote.Host)
			task.Remote.Hosts = dedupeStrings(task.Remote.Hosts)
			if task.Remote.Scheduler != nil {
				task.Remote.Scheduler.Args = dedupeStrings(task.Remote.Scheduler.Args)
			}
		}
		if task.Scheduler != nil {
			task.Scheduler.Args = dedupeStrings(task.Scheduler.Args)
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

func (c *Config) normalizeDatasets() {
	for name, dataset := range c.Datasets {
		if dataset == nil {
			continue
		}
		if strings.TrimSpace(name) == "" {
			continue
		}
		dataset.Path = strings.TrimSpace(dataset.Path)
		dataset.Manifest = strings.TrimSpace(dataset.Manifest)
		dataset.Files = dedupeStrings(dataset.Files)
		dataset.Tags = dedupeStrings(dataset.Tags)
		if dataset.Format != "" {
			dataset.Format = strings.TrimSpace(dataset.Format)
		}
	}
}

func (c *Config) validate() error {
	issues := []string{}

	validateKeyMap("vars", c.Vars, &issues)
	validateKeyMap("env", c.Env, &issues)
	validateKeyMap("seed_env", c.SeedEnv, &issues)
	if c.Seed < 0 {
		issues = append(issues, "seed must be >= 0")
	}
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
	if c.Resources != nil {
		if c.Resources.CPU < 0 {
			issues = append(issues, "resources.cpu must be >= 0")
		}
		if c.Resources.GPUs < 0 {
			issues = append(issues, "resources.gpus must be >= 0")
		}
		if c.Resources.Memory != "" {
			if _, err := parseByteSize(c.Resources.Memory); err != nil {
				issues = append(issues, fmt.Sprintf("resources.memory invalid size: %s", err))
			}
		}
		for name, limit := range c.Resources.Groups {
			if strings.TrimSpace(name) == "" {
				issues = append(issues, "resources.groups key must not be empty")
				continue
			}
			if limit <= 0 {
				issues = append(issues, fmt.Sprintf("resources.groups.%s must be > 0", name))
			}
		}
	}
	if c.Offline != nil {
		validateKeyMap("offline.env", c.Offline.Env, &issues)
	}
	if c.Experiments != nil {
		if strings.TrimSpace(c.Experiments.Dir) == "" && c.Experiments.Dir != "" {
			issues = append(issues, "experiments.dir must not be empty")
		}
		validateKeyMap("experiments.metadata", c.Experiments.Metadata, &issues)
		for i, tag := range c.Experiments.Tags {
			if strings.TrimSpace(tag) == "" {
				issues = append(issues, fmt.Sprintf("experiments.tags[%d] must not be empty", i))
			}
		}
	}
	if c.Registry != nil {
		if strings.TrimSpace(c.Registry.Type) == "" {
			issues = append(issues, "registry.type must not be empty")
		}
		if strings.TrimSpace(c.Registry.Path) == "" {
			issues = append(issues, "registry.path must not be empty")
		}
	}
	if c.Snapshot != nil && strings.TrimSpace(c.Snapshot.Path) == "" && c.Snapshot.Path != "" {
		issues = append(issues, "snapshot.path must not be empty")
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
	for name, dataset := range c.Datasets {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			issues = append(issues, "datasets key must not be empty")
			continue
		}
		if dataset == nil {
			issues = append(issues, fmt.Sprintf("datasets.%s must be an object", name))
			continue
		}
		if dataset.Path == "" && len(dataset.Files) == 0 && dataset.Manifest == "" {
			issues = append(issues, fmt.Sprintf("datasets.%s must define path, files, or manifest", name))
		}
		if strings.TrimSpace(dataset.Path) == "" && dataset.Path != "" {
			issues = append(issues, fmt.Sprintf("datasets.%s.path must not be empty", name))
		}
		if strings.TrimSpace(dataset.Manifest) == "" && dataset.Manifest != "" {
			issues = append(issues, fmt.Sprintf("datasets.%s.manifest must not be empty", name))
		}
		for i, file := range dataset.Files {
			if strings.TrimSpace(file) == "" {
				issues = append(issues, fmt.Sprintf("datasets.%s.files[%d] must not be empty", name, i))
			}
		}
		for i, tag := range dataset.Tags {
			if strings.TrimSpace(tag) == "" {
				issues = append(issues, fmt.Sprintf("datasets.%s.tags[%d] must not be empty", name, i))
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

		if len(task.Deps) == 0 && len(task.DependsOn) == 0 && !taskHasActions(task) {
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

	if task.Script != "" && len(task.Run) > 0 {
		*issues = append(*issues, fmt.Sprintf("%s.script cannot be used with %s.run", path, path))
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
	if task.RunDir != "" && strings.TrimSpace(task.RunDir) == "" {
		*issues = append(*issues, fmt.Sprintf("%s.run_dir must not be empty", path))
	}
	if task.Seed < 0 {
		*issues = append(*issues, fmt.Sprintf("%s.seed must be >= 0", path))
	}
	validateKeyMap(fmt.Sprintf("%s.seed_env", path), task.SeedEnv, issues)
	if task.Resources != nil {
		if task.Resources.CPU < 0 {
			*issues = append(*issues, fmt.Sprintf("%s.resources.cpu must be >= 0", path))
		}
		if task.Resources.GPU < 0 {
			*issues = append(*issues, fmt.Sprintf("%s.resources.gpu must be >= 0", path))
		}
		if task.Resources.Memory != "" {
			if _, err := parseByteSize(task.Resources.Memory); err != nil {
				*issues = append(*issues, fmt.Sprintf("%s.resources.memory invalid size: %s", path, err))
			}
		}
		if task.Resources.Group != "" && strings.TrimSpace(task.Resources.Group) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.resources.group must not be empty", path))
		}
	}
	if task.Group != "" && strings.TrimSpace(task.Group) == "" {
		*issues = append(*issues, fmt.Sprintf("%s.group must not be empty", path))
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
	if task.Sweep != nil && len(task.Matrix) > 0 {
		*issues = append(*issues, fmt.Sprintf("%s.sweep cannot be used with %s.matrix", path, path))
	}
	if task.Sweep != nil {
		if len(task.Sweep.Grid) == 0 && len(task.Sweep.Sample) == 0 {
			*issues = append(*issues, fmt.Sprintf("%s.sweep must define grid or sample", path))
		}
		if len(task.Sweep.Sample) > 0 && task.Sweep.Samples <= 0 {
			*issues = append(*issues, fmt.Sprintf("%s.sweep.samples must be > 0 when sample is set", path))
		}
		for key, values := range task.Sweep.Grid {
			if strings.TrimSpace(key) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.sweep.grid key must not be empty", path))
				continue
			}
			if len(values) == 0 {
				*issues = append(*issues, fmt.Sprintf("%s.sweep.grid.%s must not be empty", path, key))
			}
		}
		for key, values := range task.Sweep.Sample {
			if strings.TrimSpace(key) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.sweep.sample key must not be empty", path))
				continue
			}
			if len(values) == 0 {
				*issues = append(*issues, fmt.Sprintf("%s.sweep.sample.%s must not be empty", path, key))
			}
		}
	}
	for i, ref := range task.Datasets {
		if strings.TrimSpace(ref.Name) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.datasets[%d].name must not be empty", path, i))
		}
	}
	for i, out := range task.DatasetOutputs {
		if strings.TrimSpace(out.Name) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.dataset_outputs[%d].name must not be empty", path, i))
		}
		if strings.TrimSpace(out.Path) == "" && len(out.Files) == 0 {
			*issues = append(*issues, fmt.Sprintf("%s.dataset_outputs[%d] must define path or files", path, i))
		}
		for j, file := range out.Files {
			if strings.TrimSpace(file) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.dataset_outputs[%d].files[%d] must not be empty", path, i, j))
			}
		}
		for j, tag := range out.Tags {
			if strings.TrimSpace(tag) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.dataset_outputs[%d].tags[%d] must not be empty", path, i, j))
			}
		}
	}
	if task.Split != nil {
		if strings.TrimSpace(task.Split.Input) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.split.input must not be empty", path))
		}
		if strings.TrimSpace(task.Split.Output) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.split.output must not be empty", path))
		}
		total := task.Split.Train + task.Split.Val + task.Split.Test
		if total <= 0 {
			*issues = append(*issues, fmt.Sprintf("%s.split must define train/val/test ratios", path))
		}
		if total > 1.001 {
			*issues = append(*issues, fmt.Sprintf("%s.split ratios must sum to <= 1.0", path))
		}
		if task.Split.Seed < 0 {
			*issues = append(*issues, fmt.Sprintf("%s.split.seed must be >= 0", path))
		}
	}
	if task.Validate != nil {
		for i, item := range task.Validate.Paths {
			if strings.TrimSpace(item) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.validate.paths[%d] must not be empty", path, i))
			}
		}
		if task.Validate.MinFiles < 0 || task.Validate.MaxFiles < 0 {
			*issues = append(*issues, fmt.Sprintf("%s.validate min/max must be >= 0", path))
		}
		if task.Validate.MaxFiles > 0 && task.Validate.MinFiles > task.Validate.MaxFiles {
			*issues = append(*issues, fmt.Sprintf("%s.validate min_files must be <= max_files", path))
		}
		if task.Validate.MinSize != "" {
			if _, err := parseByteSize(task.Validate.MinSize); err != nil {
				*issues = append(*issues, fmt.Sprintf("%s.validate.min_size invalid size: %s", path, err))
			}
		}
		if task.Validate.MaxSize != "" {
			if _, err := parseByteSize(task.Validate.MaxSize); err != nil {
				*issues = append(*issues, fmt.Sprintf("%s.validate.max_size invalid size: %s", path, err))
			}
		}
	}
	if task.Stats != nil && len(task.Stats.Paths) == 0 {
		*issues = append(*issues, fmt.Sprintf("%s.stats.paths must not be empty", path))
	}
	if task.Metrics != nil {
		if len(task.Metrics.Regex) == 0 && strings.TrimSpace(task.Metrics.File) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.metrics must define regex or file", path))
		}
		if task.Metrics.Format != "" && task.Metrics.Format != "json" && task.Metrics.Format != "csv" && task.Metrics.Format != "kv" {
			*issues = append(*issues, fmt.Sprintf("%s.metrics.format must be json, csv, or kv", path))
		}
	}
	if task.Canary != nil {
		if strings.TrimSpace(task.Canary.Baseline) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.canary.baseline must not be empty", path))
		}
		if len(task.Canary.Rules) == 0 {
			*issues = append(*issues, fmt.Sprintf("%s.canary.rules must not be empty", path))
		}
		for key := range task.Canary.Rules {
			if strings.TrimSpace(key) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.canary.rules key must not be empty", path))
			}
		}
	}
	if task.Benchmark != nil {
		if task.Benchmark.Iterations <= 0 {
			*issues = append(*issues, fmt.Sprintf("%s.benchmark.iterations must be > 0", path))
		}
		if task.Benchmark.Warmup < 0 {
			*issues = append(*issues, fmt.Sprintf("%s.benchmark.warmup must be >= 0", path))
		}
	}
	if task.Experiment != nil {
		for i, tag := range task.Experiment.Tags {
			if strings.TrimSpace(tag) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.experiment.tags[%d] must not be empty", path, i))
			}
		}
		validateKeyMap(fmt.Sprintf("%s.experiment.metadata", path), task.Experiment.Metadata, issues)
	}
	if task.Sign != nil && task.Sign.Output != "" && strings.TrimSpace(task.Sign.Output) == "" {
		*issues = append(*issues, fmt.Sprintf("%s.sign.output must not be empty", path))
	}
	if task.SBOM != nil {
		if task.SBOM.Path != "" && strings.TrimSpace(task.SBOM.Path) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.sbom.path must not be empty", path))
		}
		if task.SBOM.Format != "" && task.SBOM.Format != "json" && task.SBOM.Format != "txt" {
			*issues = append(*issues, fmt.Sprintf("%s.sbom.format must be json or txt", path))
		}
	}
	if task.Checkpoint != nil {
		if len(task.Checkpoint.Paths) == 0 {
			*issues = append(*issues, fmt.Sprintf("%s.checkpoint.paths must not be empty", path))
		}
		if strings.TrimSpace(task.Checkpoint.Var) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.checkpoint.var must not be empty", path))
		}
	}
	if task.ModelCard != nil {
		if strings.TrimSpace(task.ModelCard.Path) == "" && task.ModelCard.Path != "" {
			*issues = append(*issues, fmt.Sprintf("%s.model_card.path must not be empty", path))
		}
		validateKeyMap(fmt.Sprintf("%s.model_card.metadata", path), task.ModelCard.Metadata, issues)
	}
	if task.Notebook != nil {
		if strings.TrimSpace(task.Notebook.Path) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.notebook.path must not be empty", path))
		}
		if task.Notebook.Output != "" && strings.TrimSpace(task.Notebook.Output) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.notebook.output must not be empty", path))
		}
		validateKeyMap(fmt.Sprintf("%s.notebook.parameters", path), task.Notebook.Parameters, issues)
	}
	if task.Export != nil {
		if strings.TrimSpace(task.Export.Path) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.export.path must not be empty", path))
		}
		if task.Export.Format != "" && task.Export.Format != "dir" && task.Export.Format != "zip" && task.Export.Format != "tar.gz" {
			*issues = append(*issues, fmt.Sprintf("%s.export.format must be dir, zip, or tar.gz", path))
		}
	}
	if task.Offline != nil {
		validateKeyMap(fmt.Sprintf("%s.offline.env", path), task.Offline.Env, issues)
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
		if strings.TrimSpace(task.Remote.Host) == "" && len(task.Remote.Hosts) == 0 {
			*issues = append(*issues, fmt.Sprintf("%s.remote.host or remote.hosts must be set", path))
		}
		for i, host := range task.Remote.Hosts {
			if strings.TrimSpace(host) == "" {
				*issues = append(*issues, fmt.Sprintf("%s.remote.hosts[%d] must not be empty", path, i))
			}
		}
		if task.Remote.Scheduler != nil {
			validateScheduler(fmt.Sprintf("%s.remote.scheduler", path), task.Remote.Scheduler, issues)
		}
	}
	if task.Scheduler != nil {
		validateScheduler(fmt.Sprintf("%s.scheduler", path), task.Scheduler, issues)
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

func validateScheduler(path string, sched *SchedulerSpec, issues *[]string) {
	if sched == nil {
		return
	}
	kind := strings.ToLower(strings.TrimSpace(sched.Type))
	if kind == "" {
		*issues = append(*issues, fmt.Sprintf("%s.type must not be empty", path))
	} else if kind != "slurm" && kind != "pbs" {
		*issues = append(*issues, fmt.Sprintf("%s.type must be slurm or pbs", path))
	}
	if sched.Nodes < 0 {
		*issues = append(*issues, fmt.Sprintf("%s.nodes must be >= 0", path))
	}
	if sched.GPUs < 0 {
		*issues = append(*issues, fmt.Sprintf("%s.gpus must be >= 0", path))
	}
	if sched.CPUs < 0 {
		*issues = append(*issues, fmt.Sprintf("%s.cpus must be >= 0", path))
	}
	if sched.Memory != "" {
		if _, err := parseByteSize(sched.Memory); err != nil {
			*issues = append(*issues, fmt.Sprintf("%s.memory invalid size: %s", path, err))
		}
	}
	if sched.Time != "" {
		if _, err := time.ParseDuration(sched.Time); err != nil && !strings.Contains(sched.Time, ":") {
			*issues = append(*issues, fmt.Sprintf("%s.time invalid duration: %s", path, err))
		}
	}
	for i, arg := range sched.Args {
		if strings.TrimSpace(arg) == "" {
			*issues = append(*issues, fmt.Sprintf("%s.args[%d] must not be empty", path, i))
		}
	}
}

func dedupeDatasetRefs(values []DatasetRef) []DatasetRef {
	seen := map[string]DatasetRef{}
	for _, ref := range values {
		ref.Name = strings.TrimSpace(ref.Name)
		ref.Version = strings.TrimSpace(ref.Version)
		key := fmt.Sprintf("%s:%s:%t", ref.Name, ref.Version, ref.Optional)
		if ref.Name == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = ref
	}
	out := make([]DatasetRef, 0, len(seen))
	for _, ref := range seen {
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Version < out[j].Version
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func normalizeDatasetOutputs(values []DatasetOutput) []DatasetOutput {
	out := make([]DatasetOutput, 0, len(values))
	for _, item := range values {
		item.Name = strings.TrimSpace(item.Name)
		item.Path = strings.TrimSpace(item.Path)
		item.Version = strings.TrimSpace(item.Version)
		item.Tags = dedupeStrings(item.Tags)
		item.Files = dedupeStrings(item.Files)
		out = append(out, item)
	}
	return out
}

func taskHasActions(task *Task) bool {
	if task == nil {
		return false
	}
	if task.Script != "" || len(task.Run) > 0 || len(task.Pre) > 0 || len(task.Post) > 0 || task.Docker != nil {
		return true
	}
	if task.Split != nil || task.Validate != nil || task.Stats != nil || task.Metrics != nil || task.Canary != nil {
		return true
	}
	if task.Benchmark != nil || task.Experiment != nil || task.Snapshot != nil || task.Sign != nil || task.SBOM != nil {
		return true
	}
	if task.Checkpoint != nil || task.ModelCard != nil || task.Notebook != nil || task.Export != nil {
		return true
	}
	if len(task.DatasetOutputs) > 0 {
		return true
	}
	return false
}

func parseByteSize(value string) (int64, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return 0, fmt.Errorf("empty size")
	}
	i := 0
	for i < len(trimmed) && (trimmed[i] == '.' || (trimmed[i] >= '0' && trimmed[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("invalid size: %s", value)
	}
	number := trimmed[:i]
	suffix := strings.TrimSpace(trimmed[i:])
	if suffix == "" {
		suffix = "b"
	}
	factor := int64(1)
	switch suffix {
	case "b":
		factor = 1
	case "k", "kb", "kib":
		factor = 1024
	case "m", "mb", "mib":
		factor = 1024 * 1024
	case "g", "gb", "gib":
		factor = 1024 * 1024 * 1024
	case "t", "tb", "tib":
		factor = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("invalid size suffix: %s", value)
	}
	parsed, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, err
	}
	return int64(parsed * float64(factor)), nil
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
