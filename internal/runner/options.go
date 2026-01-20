package runner

import "time"

type Options struct {
	DryRun          bool
	DryRunFormat    string
	MaxParallel     int
	ContinueOnError bool
	FailFast        bool
	Reverse         bool
	Until           string
	Explain         bool
	Profile         bool
	Progress        bool
	JSON            bool
	LogLevel        string
	Timestamp       bool
	Color           string
	Yes             bool
	EnvFile         string
	ArtifactsDir    string
	Timeout         time.Duration
	TimelinePath    string
	SummaryPath     string
	ArgsTarget      string
	Args            []string
}
