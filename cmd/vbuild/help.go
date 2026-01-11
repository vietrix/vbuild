package main

import (
	"fmt"
	"io"
	"strings"
)

func printUsage(out io.Writer) {
	if out == nil {
		return
	}
	fmt.Fprintln(out, "vbuild - fast cross-platform task runner")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  vbuild [task...]")
	fmt.Fprintln(out, "  vbuild <command> [options]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  list             List tasks")
	fmt.Fprintln(out, "  graph            Show task graph")
	fmt.Fprintln(out, "  watch            Watch files and rerun tasks")
	fmt.Fprintln(out, "  tag              Run tasks by tag")
	fmt.Fprintln(out, "  only-changed      Run tasks affected by git changes")
	fmt.Fprintln(out, "  inspect          Show resolved task definition")
	fmt.Fprintln(out, "  shell            Open a shell with task env/workdir")
	fmt.Fprintln(out, "  doctor           Check local tooling and config")
	fmt.Fprintln(out, "  clean            Remove .vbuild cache/artifacts")
	fmt.Fprintln(out, "  daemon           Run or control daemon mode")
	fmt.Fprintln(out, "  update           Self-update to GitHub release")
	fmt.Fprintln(out, "  init             Create a starter .vbuild.yml")
	fmt.Fprintln(out, "  lock             Write .vbuild.lock")
	fmt.Fprintln(out, "  version          Print version")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Global flags:")
	fmt.Fprintln(out, "  --file PATH              Config path (default .vbuild.yml)")
	fmt.Fprintln(out, "  --dry-run                Print commands without executing")
	fmt.Fprintln(out, "  --dry-run=json            Emit dry-run plan as JSON")
	fmt.Fprintln(out, "  --dry-run-format FORMAT  text|json")
	fmt.Fprintln(out, "  --max-parallel N          Limit concurrent tasks")
	fmt.Fprintln(out, "  --continue-on-error       Keep running independent tasks")
	fmt.Fprintln(out, "  --fail-fast               Stop immediately on first failure")
	fmt.Fprintln(out, "  --reverse                 Run tasks in reverse dependency order")
	fmt.Fprintln(out, "  --until TASK              Run tasks up to target in topo order")
	fmt.Fprintln(out, "  --profile                 Show critical path")
	fmt.Fprintln(out, "  --progress                Show task progress")
	fmt.Fprintln(out, "  --explain                 Explain skip reasons")
	fmt.Fprintln(out, "  --timeout DURATION        Default task timeout (e.g. 10m)")
	fmt.Fprintln(out, "  --timeline PATH           Write timeline JSON to path")
	fmt.Fprintln(out, "  --json-summary PATH       Write summary JSON ('-' for stdout)")
	fmt.Fprintln(out, "  --only-changed            Run tasks affected by git diff")
	fmt.Fprintln(out, "  --changed-base REF        Base ref for only-changed")
	fmt.Fprintln(out, "  --changed-target REF      Target ref for only-changed")
	fmt.Fprintln(out, "  --include-untracked       Include untracked files")
	fmt.Fprintln(out, "  --since TIME              Run tasks changed since time")
	fmt.Fprintln(out, "  --export-env              Write resolved env for task")
	fmt.Fprintln(out, "  --export-env-path PATH    Export env file path (default .env)")
	fmt.Fprintln(out, "  --print-vars              Print resolved vars for task")
	fmt.Fprintln(out, "  --json                    JSON logs")
	fmt.Fprintln(out, "  --log-level LEVEL         debug|info|warn|error")
	fmt.Fprintln(out, "  --timestamp               Prefix logs with timestamps")
	fmt.Fprintln(out, "  --color MODE              auto|always|never")
	fmt.Fprintln(out, "  --env-file PATH           Override env file")
	fmt.Fprintln(out, "  --artifacts-dir PATH      Override artifacts directory")
	fmt.Fprintln(out, "  --yes                     Auto-confirm prompts")
	fmt.Fprintln(out, "  --version, -V             Print version")
	fmt.Fprintln(out, "  --help, -h                Show help")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  vbuild")
	fmt.Fprintln(out, "  vbuild build test")
	fmt.Fprintln(out, "  vbuild --dry-run")
	fmt.Fprintln(out, "  vbuild --dry-run=json")
	fmt.Fprintln(out, "  vbuild only-changed --base origin/main")
}

func normalizeDryRunArgs(args []string) ([]string, string) {
	out := make([]string, 0, len(args))
	format := ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "--dry-run=") || strings.HasPrefix(arg, "-dry-run=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				format = strings.TrimSpace(parts[1])
				out = append(out, "--dry-run")
				if format != "" {
					out = append(out, "--dry-run-format="+format)
				}
				continue
			}
		}
		out = append(out, arg)
	}
	return out, format
}

func defaultDryRunFormat(override string) string {
	if override != "" {
		return override
	}
	return "text"
}
