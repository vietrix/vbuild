package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/vietrix/vbuild/internal/config"
	"github.com/vietrix/vbuild/internal/platform"
)

type taskResult struct {
	name     string
	duration time.Duration
	err      error
}

func (r *Runner) printDryRun(plan *taskPlan) {
	for _, name := range plan.order {
		task := plan.tasks[name]
		if task == nil || len(task.Run) == 0 {
			continue
		}
		r.log.Printf("==> %s\n", name)
		for _, cmd := range task.Run {
			r.log.Printf("%s\n", expandVars(cmd, r.cfg.Vars))
		}
	}
}

func (r *Runner) executePlan(plan *taskPlan) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	inDegree := make(map[string]int, len(plan.tasks))
	for name := range plan.tasks {
		inDegree[name] = len(plan.deps[name])
	}

	ready := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	done := make(chan taskResult, len(plan.tasks))
	results := make(map[string]taskResult, len(plan.tasks))
	running := 0
	var firstErr error

	startTask := func(name string) {
		running++
		task := plan.tasks[name]
		go func() {
			duration, err := r.runTask(ctx, plan, name, task)
			done <- taskResult{name: name, duration: duration, err: err}
		}()
	}

	for len(ready) > 0 && firstErr == nil {
		name := ready[0]
		ready = ready[1:]
		startTask(name)
	}

	for running > 0 {
		result := <-done
		running--
		results[result.name] = result
		if result.err != nil && firstErr == nil {
			firstErr = result.err
			cancel()
		}

		for _, dependent := range plan.dependents[result.name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}

		if firstErr == nil {
			sort.Strings(ready)
			for len(ready) > 0 {
				name := ready[0]
				ready = ready[1:]
				startTask(name)
			}
		}
	}

	r.printSummary(plan, results, time.Since(start))
	return firstErr
}

func (r *Runner) runTask(ctx context.Context, plan *taskPlan, name string, task *config.Task) (time.Duration, error) {
	start := time.Now()
	if task == nil {
		return 0, fmt.Errorf("task not found: %s", name)
	}
	if len(task.Run) == 0 {
		return 0, nil
	}

	r.log.Printf("==> %s\n", name)

	if r.dryRun {
		for _, cmd := range task.Run {
			r.log.Printf("%s\n", expandVars(cmd, r.cfg.Vars))
		}
		return 0, nil
	}

	if err := ctx.Err(); err != nil {
		return 0, err
	}

	env := mergeEnv(os.Environ(), r.cfg.Env, task.Env)
	prefixed := plan.prefixOutput || task.Parallel

	var err error
	if task.Parallel {
		err = r.runParallel(ctx, name, task.Run, env, prefixed)
	} else {
		err = r.runSequential(ctx, name, task.Run, env, prefixed)
	}
	duration := time.Since(start)

	if err != nil {
		r.log.Errorf("==> %s failed in %s\n", name, formatDuration(duration))
		return duration, err
	}

	r.log.Printf("==> %s completed in %s\n", name, formatDuration(duration))
	return duration, nil
}

func (r *Runner) printSummary(plan *taskPlan, results map[string]taskResult, total time.Duration) {
	okCount := 0
	failCount := 0
	skipCount := 0

	r.log.Printf("==> summary\n")
	for _, name := range plan.order {
		result, ok := results[name]
		if !ok {
			skipCount++
			r.log.Printf("%s skipped\n", name)
			continue
		}
		if result.err != nil {
			failCount++
			r.log.Printf("%s failed in %s\n", name, formatDuration(result.duration))
			continue
		}
		okCount++
		r.log.Printf("%s completed in %s\n", name, formatDuration(result.duration))
	}

	r.log.Printf("==> total %s (ok=%d failed=%d skipped=%d)\n", formatDuration(total), okCount, failCount, skipCount)
}

func (r *Runner) runSequential(ctx context.Context, taskName string, commands []string, env []string, prefixed bool) error {
	prefix := ""
	if prefixed {
		prefix = fmt.Sprintf("[%s] ", taskName)
	}
	for _, command := range commands {
		if err := ctx.Err(); err != nil {
			return err
		}
		command = expandVars(command, r.cfg.Vars)
		if err := r.runCommand(ctx, command, env, prefix, prefixed); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) runParallel(ctx context.Context, taskName string, commands []string, env []string, prefixed bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(commands))

	for i, command := range commands {
		cmdText := expandVars(command, r.cfg.Vars)
		prefix := ""
		if prefixed {
			prefix = fmt.Sprintf("[%s:%d] ", taskName, i+1)
		}
		wg.Add(1)
		go func(cmd, cmdPrefix string) {
			defer wg.Done()
			errCh <- r.runCommand(ctx, cmd, env, cmdPrefix, prefixed)
		}(cmdText, prefix)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var firstErr error
	for err := range errCh {
		if err == nil {
			continue
		}
		if errors.Is(err, context.Canceled) {
			continue
		}
		if firstErr == nil {
			firstErr = err
			cancel()
		}
	}

	return firstErr
}

func (r *Runner) runCommand(ctx context.Context, command string, env []string, prefix string, prefixed bool) error {
	cmd := platform.ShellCommand(ctx, command)
	platform.ConfigureCommand(cmd)
	cmd.Env = env
	cmd.Stdin = os.Stdin

	if prefixed {
		cmd.Stdout = newPrefixWriter(prefix, r.log.out, &r.log.mu)
		cmd.Stderr = newPrefixWriter(prefix, r.log.err, &r.log.mu)
	} else {
		cmd.Stdout = r.log.out
		cmd.Stderr = r.log.err
	}

	if err := cmd.Run(); err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		exitCode := 1
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return &commandError{cmd: command, code: exitCode, err: err}
	}

	return nil
}
