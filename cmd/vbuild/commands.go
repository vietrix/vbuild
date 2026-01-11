package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vietrix/vbuild/internal/config"
	"github.com/vietrix/vbuild/internal/daemon"
	"github.com/vietrix/vbuild/internal/platform"
	"github.com/vietrix/vbuild/internal/runner"
)

func runList(args []string, cfg *config.Config, opts runner.Options, jsonFlag bool, stdout, stderr io.Writer) int {
	listFlags := flag.NewFlagSet("list", flag.ContinueOnError)
	listFlags.SetOutput(stderr)
	listJSON := listFlags.Bool("json", false, "emit JSON list")
	if err := listFlags.Parse(args); err != nil {
		return 2
	}
	r := runner.New(cfg, opts, stdout, stderr)
	if *listJSON || jsonFlag {
		if err := r.ListTasksJSON(stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	r.ListTasks()
	return 0
}

func runGraph(args []string, cfg *config.Config, opts runner.Options, stdout, stderr io.Writer) int {
	graphFlags := flag.NewFlagSet("graph", flag.ContinueOnError)
	graphFlags.SetOutput(stderr)
	format := graphFlags.String("format", "dot", "graph format (dot, json)")
	all := graphFlags.Bool("all", false, "include all tasks")
	if err := graphFlags.Parse(args); err != nil {
		return 2
	}
	targets := graphFlags.Args()
	if !*all && len(targets) == 0 {
		targets = []string{"default"}
	}
	r := runner.New(cfg, opts, stdout, stderr)
	if err := r.Graph(targets, *format, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runWatch(args []string, cfg *config.Config, opts runner.Options, stdout, stderr io.Writer) int {
	watchFlags := flag.NewFlagSet("watch", flag.ContinueOnError)
	watchFlags.SetOutput(stderr)
	interval := watchFlags.Duration("interval", time.Second, "polling interval")
	debounce := watchFlags.Duration("debounce", 300*time.Millisecond, "debounce window")
	if err := watchFlags.Parse(args); err != nil {
		return 2
	}
	taskName := "default"
	if args := watchFlags.Args(); len(args) > 0 {
		taskName = args[0]
	}
	r := runner.New(cfg, opts, stdout, stderr)
	if err := r.Watch(taskName, *interval, *debounce); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runTag(args []string, cfg *config.Config, opts runner.Options, stdout, stderr io.Writer) int {
	tagFlags := flag.NewFlagSet("tag", flag.ContinueOnError)
	tagFlags.SetOutput(stderr)
	if err := tagFlags.Parse(args); err != nil {
		return 2
	}
	parsed := tagFlags.Args()
	if len(parsed) == 0 {
		fmt.Fprintln(stderr, "tag name is required")
		return 2
	}
	r := runner.New(cfg, opts, stdout, stderr)
	targets := r.TasksByTag(parsed[0])
	if len(targets) == 0 {
		fmt.Fprintf(stderr, "no tasks found for tag %q\n", parsed[0])
		return 1
	}
	if err := r.RunTargets(targets); err != nil {
		fmt.Fprintln(stderr, err)
		return runner.ExitCode(err)
	}
	return 0
}

func runDoctor(cfg *config.Config, stdout, stderr io.Writer) int {
	type check struct {
		name     string
		required bool
	}
	checks := []check{
		{name: "git", required: false},
		{name: "go", required: false},
	}

	needsDocker := false
	needsSSH := false
	for _, task := range cfg.Tasks {
		if task == nil {
			continue
		}
		if task.Docker != nil {
			needsDocker = true
		}
		if task.Remote != nil {
			needsSSH = true
		}
	}
	if needsDocker {
		checks = append(checks, check{name: "docker", required: true})
	}
	if needsSSH {
		checks = append(checks, check{name: "ssh", required: true})
	}

	if _, err := exec.LookPath(platform.BinaryName()); err != nil {
		fmt.Fprintf(stdout, "warn: vbuild not on PATH\n")
	}

	ok := true
	for _, item := range checks {
		if _, err := exec.LookPath(item.name); err != nil {
			level := "warn"
			if item.required {
				level = "error"
				ok = false
			}
			fmt.Fprintf(stdout, "%s: %s not found\n", level, item.name)
			continue
		}
		fmt.Fprintf(stdout, "ok: %s\n", item.name)
	}

	if cfg.Path == "" {
		fmt.Fprintln(stdout, "warn: config path not set")
	} else {
		fmt.Fprintf(stdout, "ok: config %s\n", cfg.Path)
	}

	if !ok {
		return 1
	}
	return 0
}

func runClean(cfg *config.Config, opts runner.Options, stdout, stderr io.Writer) int {
	r := runner.New(cfg, opts, stdout, stderr)
	if err := r.Clean(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "removed .vbuild")
	return 0
}

func runShell(args []string, cfg *config.Config, opts runner.Options, stdout, stderr io.Writer) int {
	shellFlags := flag.NewFlagSet("shell", flag.ContinueOnError)
	shellFlags.SetOutput(stderr)
	if err := shellFlags.Parse(args); err != nil {
		return 2
	}
	taskName := "default"
	if args := shellFlags.Args(); len(args) > 0 {
		taskName = args[0]
	}
	r := runner.New(cfg, opts, stdout, stderr)
	if err := r.OpenShell(taskName); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runInspect(args []string, cfg *config.Config, opts runner.Options, stdout, stderr io.Writer) int {
	inspectFlags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	inspectFlags.SetOutput(stderr)
	if err := inspectFlags.Parse(args); err != nil {
		return 2
	}
	taskName := "default"
	if args := inspectFlags.Args(); len(args) > 0 {
		taskName = args[0]
	}
	r := runner.New(cfg, opts, stdout, stderr)
	if err := r.Inspect(taskName, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runOnlyChanged(args []string, cfg *config.Config, opts runner.Options, stdout, stderr io.Writer) int {
	changedFlags := flag.NewFlagSet("only-changed", flag.ContinueOnError)
	changedFlags.SetOutput(stderr)
	base := changedFlags.String("base", "HEAD", "git base ref")
	target := changedFlags.String("target", "", "git target ref")
	list := changedFlags.Bool("list", false, "list matching tasks")
	jsonOut := changedFlags.Bool("json", false, "emit JSON list")
	includeUntracked := changedFlags.Bool("include-untracked", true, "include untracked files")
	if err := changedFlags.Parse(args); err != nil {
		return 2
	}
	targets := changedFlags.Args()

	r := runner.New(cfg, opts, stdout, stderr)
	selected, err := r.ChangedTargets(targets, runner.ChangedOptions{
		BaseRef:          *base,
		TargetRef:        *target,
		IncludeUntracked: *includeUntracked,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *list || *jsonOut {
		if *jsonOut {
			data, _ := json.MarshalIndent(selected, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			for _, name := range selected {
				fmt.Fprintln(stdout, name)
			}
		}
		return 0
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

func runDaemon(args []string, cfg *config.Config, opts runner.Options, version string, stdout, stderr io.Writer) int {
	daemonFlags := flag.NewFlagSet("daemon", flag.ContinueOnError)
	daemonFlags.SetOutput(stderr)
	addr := daemonFlags.String("addr", "127.0.0.1:8377", "listen address")
	token := daemonFlags.String("token", "", "auth token")
	if err := daemonFlags.Parse(args); err != nil {
		return 2
	}
	rest := daemonFlags.Args()
	sub := "serve"
	if len(rest) > 0 {
		sub = rest[0]
		rest = rest[1:]
	}

	infoPath := daemonInfoPath(cfg)

	switch sub {
	case "serve":
		if *token == "" {
			*token = randomToken()
		}
		ctx := context.Background()
		if err := daemon.Serve(ctx, cfg, opts, version, *addr, *token, infoPath); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "start":
		if *token == "" {
			*token = randomToken()
		}
		if err := startDaemon(cfg, *addr, *token, stdout, stderr); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "daemon started on %s\n", *addr)
		return 0
	case "status":
		return daemonStatus(infoPath, stdout, stderr)
	case "stop":
		return daemonStop(infoPath, stdout, stderr)
	case "run":
		return daemonRun(infoPath, rest, stdout, stderr)
	case "list":
		return daemonList(infoPath, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown daemon command: %s\n", sub)
		return 2
	}
}

func daemonInfoPath(cfg *config.Config) string {
	root := "."
	if cfg != nil && cfg.Path != "" && !strings.HasPrefix(cfg.Path, "http://") && !strings.HasPrefix(cfg.Path, "https://") {
		root = filepath.Dir(cfg.Path)
	}
	return filepath.Join(root, ".vbuild", "daemon.json")
}

func startDaemon(cfg *config.Config, addr, token string, stdout, stderr io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"daemon", "serve", "--addr", addr, "--token", token}
	if cfg != nil && cfg.Path != "" {
		args = append(args, "--file", cfg.Path)
	}

	cmd := exec.Command(exe, args...)
	logPath := filepath.Join(filepath.Dir(daemonInfoPath(cfg)), "daemon.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	platform.ConfigureCommand(cmd)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}

	infoPath := daemonInfoPath(cfg)
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(infoPath); err == nil {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("daemon did not write info file")
}

func daemonStatus(infoPath string, stdout, stderr io.Writer) int {
	info, err := daemon.LoadInfo(infoPath)
	if err != nil {
		fmt.Fprintln(stderr, "daemon not running")
		return 1
	}
	data, err := daemonRequest(info, "GET", "/status", nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	stdout.Write(data)
	return 0
}

func daemonStop(infoPath string, stdout, stderr io.Writer) int {
	info, err := daemon.LoadInfo(infoPath)
	if err != nil {
		fmt.Fprintln(stderr, "daemon not running")
		return 1
	}
	if _, err := daemonRequest(info, "POST", "/shutdown", map[string]string{}); err != nil {
		if info.PID != 0 {
			if proc, err := os.FindProcess(info.PID); err == nil {
				_ = proc.Kill()
			}
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "daemon stopped")
	return 0
}

func daemonRun(infoPath string, tasks []string, stdout, stderr io.Writer) int {
	info, err := daemon.LoadInfo(infoPath)
	if err != nil {
		fmt.Fprintln(stderr, "daemon not running")
		return 1
	}
	req := daemon.RunRequest{Targets: tasks}
	data, err := daemonRequest(info, "POST", "/run", req)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var resp daemon.RunResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		stdout.Write(data)
		return 0
	}
	if resp.Stdout != "" {
		fmt.Fprint(stdout, resp.Stdout)
	}
	if resp.Stderr != "" {
		fmt.Fprint(stderr, resp.Stderr)
	}
	if resp.ExitCode != 0 {
		return resp.ExitCode
	}
	return 0
}

func daemonList(infoPath string, stdout, stderr io.Writer) int {
	info, err := daemon.LoadInfo(infoPath)
	if err != nil {
		fmt.Fprintln(stderr, "daemon not running")
		return 1
	}
	data, err := daemonRequest(info, "GET", "/tasks", nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	stdout.Write(data)
	return 0
}

func daemonRequest(info *daemon.DaemonInfo, method, path string, payload interface{}) ([]byte, error) {
	if info == nil || info.Addr == "" {
		return nil, fmt.Errorf("daemon info missing address")
	}
	url := "http://" + info.Addr + path
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if info.Token != "" {
		req.Header.Set("X-VBuild-Token", info.Token)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("daemon request failed: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func randomToken() string {
	seed := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	sum := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("%x", sum[:])
}
