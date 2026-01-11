package runner

import "time"

type Options struct {
	DryRun          bool
	DryRunFormat    string
	MaxParallel     int
	ContinueOnError bool
	Explain         bool
	Profile         bool
	JSON            bool
	LogLevel        string
	Timestamp       bool
	Color           string
	Yes             bool
	EnvFile         string
	ArtifactsDir    string
	Timeout         time.Duration
	TimelinePath    string
}
