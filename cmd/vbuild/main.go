package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vietrix/vbuild/internal/config"
	"github.com/vietrix/vbuild/internal/runner"
	"github.com/vietrix/vbuild/internal/update"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	args, dryRunOverride := normalizeDryRunArgs(args)
	fs := flag.NewFlagSet("vbuild", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("file", ".vbuild.yml", "config path")
	dryRun := fs.Bool("dry-run", false, "print commands without executing")
	dryRunFormat := fs.String("dry-run-format", defaultDryRunFormat(dryRunOverride), "dry-run output format (text|json)")
	versionFlag := fs.Bool("version", false, "print version")
	versionShort := fs.Bool("V", false, "print version")
	helpFlag := fs.Bool("help", false, "show help")
	helpShort := fs.Bool("h", false, "show help")
	jsonFlag := fs.Bool("json", false, "emit JSON logs")
	logLevel := fs.String("log-level", "info", "log level (debug, info, warn, error)")
	profile := fs.Bool("profile", false, "print critical path and task timings")
	maxParallel := fs.Int("max-parallel", 0, "limit max concurrent tasks")
	continueOnError := fs.Bool("continue-on-error", false, "continue executing independent tasks after failures")
	failFast := fs.Bool("fail-fast", false, "stop immediately on first failure")
	reverse := fs.Bool("reverse", false, "run tasks in reverse dependency order")
	until := fs.String("until", "", "run tasks up to a target in topo order")
	yes := fs.Bool("yes", false, "auto-confirm prompts")
	envFile := fs.String("env-file", "", "override env file path")
	strictLock := fs.Bool("strict-lock", false, "fail if .vbuild.lock version mismatches")
	ignoreLock := fs.Bool("ignore-lock", false, "ignore .vbuild.lock")
	explain := fs.Bool("explain", false, "explain skip reasons")
	timestamp := fs.Bool("timestamp", false, "include timestamps in logs")
	color := fs.String("color", "auto", "color output (auto, always, never)")
	timeout := fs.Duration("timeout", 0, "default task timeout")
	timeline := fs.String("timeline", "", "write timeline JSON to path")
	progress := fs.Bool("progress", false, "show task progress")
	jsonSummary := fs.String("json-summary", "", "write summary JSON to path ('-' for stdout)")
	artifactsDir := fs.String("artifacts-dir", "", "override artifacts directory")
	onlyChanged := fs.Bool("only-changed", false, "run only tasks affected by git changes")
	changedBase := fs.String("changed-base", "HEAD", "git base ref for only-changed")
	changedTarget := fs.String("changed-target", "", "git target ref for only-changed")
	includeUntracked := fs.Bool("include-untracked", true, "include untracked files for only-changed")
	since := fs.String("since", "", "run tasks changed since timestamp (RFC3339 or duration)")
	exportEnv := fs.Bool("export-env", false, "write resolved env to .env")
	exportEnvPath := fs.String("export-env-path", ".env", "export env file path")
	printVars := fs.Bool("print-vars", false, "print resolved vars")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *helpFlag || *helpShort {
		printUsage(stdout)
		return 0
	}
	if *versionFlag || *versionShort {
		fmt.Fprintln(stdout, version)
		return 0
	}

	rest := fs.Args()
	if len(rest) > 0 {
		switch rest[0] {
		case "help":
			printUsage(stdout)
			return 0
		case "version":
			fmt.Fprintln(stdout, version)
			return 0
		case "update":
			return runUpdate(rest[1:], stdout, stderr)
		case "init":
			return runInit(*configPath, stdout, stderr)
		case "lock":
			return runLock(*configPath, stdout, stderr)
		}
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := checkLock(*configPath, version, cfg.Hash, *strictLock, *ignoreLock, stderr); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if *onlyChanged && *since != "" {
		fmt.Fprintln(stderr, "--only-changed and --since cannot be used together")
		return 2
	}

	opts := runner.Options{
		DryRun:          *dryRun,
		DryRunFormat:    *dryRunFormat,
		MaxParallel:     *maxParallel,
		ContinueOnError: *continueOnError,
		FailFast:        *failFast || cfg.FailFast,
		Reverse:         *reverse,
		Until:           *until,
		Explain:         *explain,
		Profile:         *profile,
		Progress:        *progress,
		JSON:            *jsonFlag,
		LogLevel:        *logLevel,
		Timestamp:       *timestamp,
		Color:           *color,
		Yes:             *yes,
		EnvFile:         *envFile,
		ArtifactsDir:    *artifactsDir,
		Timeout:         *timeout,
		TimelinePath:    *timeline,
		SummaryPath:     *jsonSummary,
	}

	if *exportEnv || *printVars {
		taskName := "default"
		if len(rest) > 0 {
			taskName = rest[0]
		}
		resolved, err := resolveSingleTarget(cfg, taskName)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		r := runner.New(cfg, opts, stdout, stderr)
		if *exportEnv {
			if err := r.ExportEnv(resolved, *exportEnvPath); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			return 0
		}
		if err := r.PrintVars(resolved, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	if len(rest) > 0 {
		switch rest[0] {
		case "list":
			return runList(rest[1:], cfg, opts, *jsonFlag, stdout, stderr)
		case "graph":
			return runGraph(rest[1:], cfg, opts, stdout, stderr)
		case "watch":
			return runWatch(rest[1:], cfg, opts, stdout, stderr)
		case "tag":
			return runTag(rest[1:], cfg, opts, stdout, stderr)
		case "doctor":
			return runDoctor(cfg, stdout, stderr)
		case "clean":
			return runClean(cfg, opts, stdout, stderr)
		case "shell":
			return runShell(rest[1:], cfg, opts, stdout, stderr)
		case "inspect":
			return runInspect(rest[1:], cfg, opts, stdout, stderr)
		case "daemon":
			return runDaemon(rest[1:], cfg, opts, version, stdout, stderr)
		case "only-changed":
			return runOnlyChanged(rest[1:], cfg, opts, stdout, stderr)
		}
	}

	targets := []string{"default"}
	if len(rest) > 0 {
		targets = rest
	}
	resolvedTargets, err := resolveTargets(cfg, targets)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if opts.Until != "" {
		untilTarget, err := resolveSingleTarget(cfg, opts.Until)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		opts.Until = untilTarget
	}

	r := runner.New(cfg, opts, stdout, stderr)
	if *onlyChanged {
		filter := []string{}
		if len(rest) > 0 {
			filter = resolvedTargets
		}
		selected, err := r.ChangedTargets(filter, runner.ChangedOptions{
			BaseRef:          *changedBase,
			TargetRef:        *changedTarget,
			IncludeUntracked: *includeUntracked,
		})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if len(selected) == 0 {
			fmt.Fprintln(stdout, "no tasks matched changes")
			return 0
		}
		if err := r.RunTargets(selected); err != nil {
			fmt.Fprintln(stderr, err)
			return runner.ExitCode(err)
		}
		return 0
	}

	if *since != "" {
		sinceTime, err := parseSince(*since)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		filter := []string{}
		if len(rest) > 0 {
			filter = resolvedTargets
		}
		selected, err := r.ChangedSince(filter, sinceTime)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if len(selected) == 0 {
			fmt.Fprintln(stdout, "no tasks matched changes")
			return 0
		}
		if err := r.RunTargets(selected); err != nil {
			fmt.Fprintln(stderr, err)
			return runner.ExitCode(err)
		}
		return 0
	}

	if err := r.RunTargets(resolvedTargets); err != nil {
		fmt.Fprintln(stderr, err)
		return runner.ExitCode(err)
	}
	return 0
}

func runUpdate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	to := fs.String("to", "", "update to specific version")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := update.Run(update.Options{ToVersion: *to, Out: stdout}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type lockFile struct {
	Version    string   `json:"version"`
	Config     string   `json:"config"`
	ConfigHash string   `json:"config_hash"`
	Sources    []string `json:"sources,omitempty"`
	Timestamp  string   `json:"timestamp"`
}

func runInit(configPath string, stdout, stderr io.Writer) int {
	if _, err := os.Stat(configPath); err == nil {
		fmt.Fprintln(stderr, "config file already exists")
		return 1
	}
	template := `workflow: "vbuild starter"

vars:
  APP: app
  VERSION: dev

env:
  CGO_ENABLED: "0"

tasks:
  default:
    desc: format and test
    deps: [format, test]

  format:
    run:
      - gofmt -w .

  test:
    run:
      - go test ./...

  build:
    desc: build binary
    run:
      - go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version={{VERSION}}" -o {{APP}} ./cmd/{{APP}}
`
	if err := os.WriteFile(configPath, []byte(template), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "created %s\n", configPath)
	return 0
}

func runLock(configPath string, stdout, stderr io.Writer) int {
	lockPath := filepath.Join(filepath.Dir(configPath), ".vbuild.lock")
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	payload := lockFile{
		Version:    version,
		Config:     cfg.Path,
		ConfigHash: cfg.Hash,
		Sources:    append([]string(nil), cfg.Sources...),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(lockPath, data, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", lockPath)
	return 0
}

func checkLock(configPath, currentVersion, configHash string, strict, ignore bool, stderr io.Writer) error {
	if ignore {
		return nil
	}
	lockPath := filepath.Join(filepath.Dir(configPath), ".vbuild.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var lock lockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return fmt.Errorf("parse lock file: %w", err)
	}
	if lock.Version != "" && lock.Version != currentVersion {
		message := fmt.Sprintf("lock version mismatch: %s (lock) != %s (current)", lock.Version, currentVersion)
		if strict {
			return fmt.Errorf(message)
		}
		fmt.Fprintln(stderr, message)
	}
	if lock.ConfigHash != "" && configHash != "" && lock.ConfigHash != configHash {
		message := fmt.Sprintf("lock config hash mismatch: %s (lock) != %s (current)", lock.ConfigHash, configHash)
		if strict {
			return fmt.Errorf(message)
		}
		fmt.Fprintln(stderr, message)
	}
	return nil
}

func resolveTargets(cfg *config.Config, targets []string) ([]string, error) {
	if len(targets) == 0 {
		return targets, nil
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, name := range targets {
		resolved, err := resolveTarget(cfg, name)
		if err != nil {
			return nil, err
		}
		for _, item := range resolved {
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	return out, nil
}

func resolveSingleTarget(cfg *config.Config, name string) (string, error) {
	resolved, err := resolveTarget(cfg, name)
	if err != nil {
		return "", err
	}
	if len(resolved) == 0 {
		return "", fmt.Errorf("unknown task: %s", name)
	}
	if len(resolved) > 1 {
		return "", fmt.Errorf("task %s expands to multiple targets", name)
	}
	return resolved[0], nil
}

func resolveTarget(cfg *config.Config, name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("task name is required")
	}
	if cfg.Aliases != nil {
		if actual, ok := cfg.Aliases[name]; ok {
			name = actual
		}
	}
	if _, ok := cfg.Tasks[name]; ok {
		return []string{name}, nil
	}
	group := strings.TrimSuffix(name, ":")
	prefix := group + ":"
	groupMatches := []string{}
	for taskName := range cfg.Tasks {
		if strings.HasPrefix(taskName, prefix) {
			groupMatches = append(groupMatches, taskName)
		}
	}
	if len(groupMatches) > 0 {
		sort.Strings(groupMatches)
		return groupMatches, nil
	}
	return nil, fmt.Errorf("unknown task: %s", name)
}

func parseSince(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("since value is empty")
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, nil
	}
	if dur, err := time.ParseDuration(value); err == nil {
		return time.Now().Add(-dur), nil
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0), nil
	}
	return time.Time{}, fmt.Errorf("invalid since value: %s", value)
}
