package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vietrix/vbuild/internal/config"
	"github.com/vietrix/vbuild/internal/runner"
	"github.com/vietrix/vbuild/internal/update"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("vbuild", flag.ContinueOnError)
	fs.SetOutput(stderr)

	configPath := fs.String("file", ".vbuild.yml", "config path")
	dryRun := fs.Bool("dry-run", false, "print commands without executing")
	versionFlag := fs.Bool("version", false, "print version")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *versionFlag {
		fmt.Fprintln(stdout, version)
		return 0
	}

	rest := fs.Args()
	if len(rest) > 0 && rest[0] == "update" {
		return runUpdate(rest[1:], stdout, stderr)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if len(rest) > 0 && rest[0] == "list" {
		runner.New(cfg, *dryRun, stdout, stderr).ListTasks()
		return 0
	}

	taskName := "default"
	if len(rest) > 0 {
		taskName = rest[0]
	}

	r := runner.New(cfg, *dryRun, stdout, stderr)
	if err := r.Run(taskName); err != nil {
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
