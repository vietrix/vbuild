package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vietrix/vbuild/internal/config"
	"github.com/vietrix/vbuild/internal/platform"
)

type taskStatus string

const (
	statusOK       taskStatus = "ok"
	statusFailed   taskStatus = "failed"
	statusSkipped  taskStatus = "skipped"
	statusUpToDate taskStatus = "up-to-date"
	statusCanceled taskStatus = "canceled"
)

type taskResult struct {
	name     string
	duration time.Duration
	status   taskStatus
	err      error
	reason   string
}

type commandOptions struct {
	env          []string
	vars         map[string]string
	prefixed     bool
	retries      int
	backoff      time.Duration
	jitter       time.Duration
	timeout      time.Duration
	shell        string
	workdir      string
	secrets      []string
	retryCodes   []int
	retryRegex   []string
	retrySignals []string
	limits       *config.ResourceLimits
	remote       *config.RemoteSpec
	silent       bool
	capture      *captureSpec
}

func (r *Runner) printDryRun(plan *taskPlan) {
	if strings.EqualFold(r.opts.DryRunFormat, "json") {
		r.printDryRunJSON(plan)
		return
	}
	for _, name := range plan.order {
		task := plan.tasks[name]
		if task == nil {
			continue
		}
		vars := r.taskVars(plan, name, task)
		commands := r.taskCommands(task, vars)
		if len(commands) == 0 {
			continue
		}
		r.log.Printf("==> %s\n", name)
		for _, cmd := range commands {
			r.log.Printf("%s\n", expandVars(cmd, vars))
		}
	}
}

type dryRunTask struct {
	Name     string   `json:"name"`
	Deps     []string `json:"deps,omitempty"`
	Commands []string `json:"commands,omitempty"`
	Workdir  string   `json:"workdir,omitempty"`
	Shell    string   `json:"shell,omitempty"`
	Parallel bool     `json:"parallel,omitempty"`
}

type dryRunPayload struct {
	Tasks []dryRunTask `json:"tasks"`
	Order []string     `json:"order"`
}

func (r *Runner) printDryRunJSON(plan *taskPlan) {
	tasks := []dryRunTask{}
	for _, name := range plan.order {
		task := plan.tasks[name]
		if task == nil {
			continue
		}
		vars := r.taskVars(plan, name, task)
		commands := r.taskCommands(task, vars)
		if len(commands) == 0 {
			continue
		}
		expanded := make([]string, 0, len(commands))
		for _, cmd := range commands {
			expanded = append(expanded, expandVars(cmd, vars))
		}
		entry := dryRunTask{
			Name:     name,
			Deps:     append([]string(nil), plan.deps[name]...),
			Commands: expanded,
			Workdir:  expandVars(task.Workdir, vars),
			Shell:    task.Shell,
			Parallel: task.Parallel,
		}
		tasks = append(tasks, entry)
	}
	payload := dryRunPayload{Tasks: tasks, Order: append([]string(nil), plan.order...)}
	enc := json.NewEncoder(r.log.out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
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
	cancelOnError := !r.opts.ContinueOnError || r.opts.FailFast

	progress := newProgressTracker(len(plan.tasks), r.log.err, r.opts.Progress && !r.opts.JSON)
	if progress.Enabled() {
		progress.Render()
	}

	var sem chan struct{}
	if r.opts.MaxParallel > 0 {
		sem = make(chan struct{}, r.opts.MaxParallel)
	}

	startTask := func(name string) {
		if depsFailed(name, plan, results) {
			done <- taskResult{name: name, status: statusSkipped, reason: "dependency failed"}
			return
		}
		if ctx.Err() != nil && cancelOnError {
			done <- taskResult{name: name, status: statusCanceled, reason: "canceled"}
			return
		}
		task := plan.tasks[name]
		running++
		if sem != nil {
			sem <- struct{}{}
		}
		go func() {
			result := r.runTask(ctx, plan, name, task)
			if sem != nil {
				<-sem
			}
			done <- result
		}()
	}

	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		startTask(name)
	}

	for running > 0 {
		result := <-done
		running--
		results[result.name] = result

		if result.status == statusFailed && firstErr == nil {
			firstErr = result.err
		}
		if result.status == statusFailed && cancelOnError {
			if r.opts.FailFast {
				cancel()
			} else if task, ok := plan.tasks[result.name]; !ok || !task.ContinueOnError {
				cancel()
			}
		}

		if progress.Enabled() {
			progress.Update(result)
		}

		for _, dependent := range plan.dependents[result.name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}

		if ctx.Err() == nil || !cancelOnError {
			sort.Strings(ready)
			for len(ready) > 0 {
				name := ready[0]
				ready = ready[1:]
				startTask(name)
			}
		}
	}

	if progress.Enabled() {
		progress.Finish()
	}
	r.printSummary(plan, results, time.Since(start))
	return firstErr
}

func depsFailed(name string, plan *taskPlan, results map[string]taskResult) bool {
	for _, dep := range plan.deps[name] {
		result, ok := results[dep]
		if !ok {
			continue
		}
		if !result.successForDeps() {
			return true
		}
	}
	return false
}

func (result taskResult) successForDeps() bool {
	return result.status == statusOK || result.status == statusUpToDate
}

func (r *Runner) runTask(ctx context.Context, plan *taskPlan, name string, task *config.Task) taskResult {
	start := time.Now()
	if task == nil {
		return taskResult{name: name, status: statusFailed, err: fmt.Errorf("task not found: %s", name)}
	}
	r.trace.TaskStart(name)
	endTrace := func(status taskStatus, duration time.Duration) {
		r.trace.TaskEnd(name, string(status), duration)
	}

	vars := r.taskVars(plan, name, task)
	envMap := r.taskEnv(vars, task)
	env := envMapToList(envMap)

	if task.When != "" {
		ok, err := evalCondition(task.When, vars, envMap)
		if err != nil {
			endTrace(statusFailed, time.Since(start))
			return taskResult{name: name, status: statusFailed, err: err}
		}
		if !ok {
			r.log.Printf("==> %s skipped (condition)\n", name)
			endTrace(statusSkipped, time.Since(start))
			return taskResult{name: name, status: statusSkipped, reason: "condition"}
		}
	}
	if task.OnlyOn != nil {
		if !matchOnlyOn(task.OnlyOn, r.gitVars) {
			r.log.Printf("==> %s skipped (only_on)\n", name)
			endTrace(statusSkipped, time.Since(start))
			return taskResult{name: name, status: statusSkipped, reason: "only_on"}
		}
	}

	if err := ctx.Err(); err != nil {
		endTrace(statusCanceled, time.Since(start))
		return taskResult{name: name, status: statusCanceled, err: err}
	}

	if task.IfMissing {
		missing, err := r.outputsMissing(task, vars)
		if err != nil {
			endTrace(statusFailed, time.Since(start))
			return taskResult{name: name, status: statusFailed, err: err}
		}
		if !missing {
			r.log.Printf("==> %s skipped (outputs exist)\n", name)
			r.recordOutputs(name, task, vars)
			endTrace(statusUpToDate, time.Since(start))
			return taskResult{name: name, status: statusUpToDate, reason: "outputs exist"}
		}
	}

	if shouldSkip, reason := r.cacheSkip(name, task, vars); shouldSkip {
		r.log.Printf("==> %s skipped (%s)\n", name, reason)
		r.recordOutputs(name, task, vars)
		endTrace(statusUpToDate, time.Since(start))
		return taskResult{name: name, status: statusUpToDate, reason: reason}
	}

	if task.Confirm != "" && !r.opts.Yes {
		prompt := expandVars(task.Confirm, vars)
		if !r.confirmTask(prompt) {
			r.log.Printf("==> %s skipped (confirmation)\n", name)
			endTrace(statusSkipped, time.Since(start))
			return taskResult{name: name, status: statusSkipped, reason: "confirmation"}
		}
	}

	if err := r.checkRequirements(task.Require); err != nil {
		endTrace(statusFailed, time.Since(start))
		return taskResult{name: name, status: statusFailed, err: err}
	}

	r.log.Printf("==> %s\n", name)

	if err := r.runPlugins(ctx, "task_start", name, "running", 0); err != nil {
		endTrace(statusFailed, time.Since(start))
		return taskResult{name: name, status: statusFailed, err: err}
	}

	commands := r.taskCommands(task, vars)
	if len(commands) == 0 && len(task.Pre) == 0 && len(task.Post) == 0 {
		duration := time.Since(start)
		r.recordOutputs(name, task, vars)
		r.log.Printf("==> %s completed in %s\n", name, formatDuration(duration))
		_ = r.runPlugins(ctx, "task_end", name, "ok", duration)
		endTrace(statusOK, duration)
		return taskResult{name: name, duration: duration, status: statusOK}
	}

	prefixed := plan.prefixOutput || task.Parallel
	retries := effectiveRetries(task, r.cfg)
	backoff := parseDuration(task.Backoff)
	jitter := parseDuration(task.Jitter)
	if backoff == 0 && r.cfg != nil && r.cfg.Defaults != nil {
		backoff = parseDuration(r.cfg.Defaults.Backoff)
	}
	if jitter == 0 && r.cfg != nil && r.cfg.Defaults != nil {
		jitter = parseDuration(r.cfg.Defaults.Jitter)
	}
	timeout := parseDuration(task.Timeout)
	if timeout == 0 && r.opts.Timeout > 0 {
		timeout = r.opts.Timeout
	}
	if timeout == 0 && r.cfg != nil && r.cfg.Defaults != nil {
		timeout = parseDuration(r.cfg.Defaults.Timeout)
	}
	if timeout == 0 && r.cfg != nil {
		timeout = parseDuration(r.cfg.Timeout)
	}

	workdir := expandVars(task.Workdir, vars)
	if workdir == "" && r.cfg != nil && r.cfg.Defaults != nil {
		workdir = expandVars(r.cfg.Defaults.Workdir, vars)
	}
	if task.Remote != nil && task.Remote.Workdir != "" {
		workdir = expandVars(task.Remote.Workdir, vars)
	}
	if task.Remote == nil {
		workdir = r.resolvePath(workdir)
	}
	shell := task.Shell
	if shell == "" && r.cfg != nil && r.cfg.Defaults != nil {
		shell = r.cfg.Defaults.Shell
	}
	secrets := r.taskSecrets(task, envMap)
	if task.Isolate && task.Remote != nil {
		runErr := fmt.Errorf("task %s cannot use isolate with remote", name)
		_ = r.runPlugins(ctx, "task_end", name, "failed", time.Since(start))
		endTrace(statusFailed, time.Since(start))
		return taskResult{name: name, duration: time.Since(start), status: statusFailed, err: runErr}
	}

	var sandbox *sandboxContext
	if task.Isolate {
		ctxSandbox, err := r.prepareSandbox(name, task, vars, workdir)
		if err != nil {
			_ = r.runPlugins(ctx, "task_end", name, "failed", time.Since(start))
			endTrace(statusFailed, time.Since(start))
			return taskResult{name: name, duration: time.Since(start), status: statusFailed, err: err}
		}
		sandbox = ctxSandbox
		defer sandbox.cleanup()
		workdir = sandbox.workdir
	}

	captureSpec, err := r.prepareCapture(task, vars)
	if err != nil {
		_ = r.runPlugins(ctx, "task_end", name, "failed", time.Since(start))
		endTrace(statusFailed, time.Since(start))
		return taskResult{name: name, duration: time.Since(start), status: statusFailed, err: err}
	}
	if captureSpec != nil {
		defer captureSpec.Close()
	}

	cmdOpts := commandOptions{
		env:          env,
		vars:         vars,
		prefixed:     prefixed,
		retries:      retries,
		backoff:      backoff,
		jitter:       jitter,
		timeout:      timeout,
		shell:        shell,
		workdir:      workdir,
		secrets:      secrets,
		retryCodes:   task.RetryOnExitCodes,
		retryRegex:   task.RetryOnRegex,
		retrySignals: task.RetryOnSignal,
		limits:       task.Limits,
		remote:       task.Remote,
		silent:       task.Silent,
		capture:      captureSpec,
	}

	runErr := r.runCommands(ctx, name, task.Pre, cmdOpts, false)
	if runErr == nil {
		if task.Parallel {
			runErr = r.runParallel(ctx, name, commands, cmdOpts)
		} else {
			runErr = r.runSequential(ctx, name, commands, cmdOpts)
		}
	}
	if runErr == nil {
		runErr = r.runCommands(ctx, name, task.Post, cmdOpts, true)
	}

	duration := time.Since(start)
	if runErr != nil {
		if task.AllowFailure {
			r.log.Printf("==> %s failed (allowed) in %s\n", name, formatDuration(duration))
			_ = r.runPlugins(ctx, "task_end", name, "allowed_failure", duration)
			endTrace(statusOK, duration)
			return taskResult{name: name, duration: duration, status: statusOK, reason: "allowed failure"}
		}
		r.log.Errorf("==> %s failed in %s\n", name, formatDuration(duration))
		_ = r.runPlugins(ctx, "task_end", name, "failed", duration)
		endTrace(statusFailed, duration)
		return taskResult{name: name, duration: duration, status: statusFailed, err: runErr}
	}

	if sandbox != nil {
		if err := r.restoreSandboxOutputs(task, vars, sandbox.root); err != nil {
			r.log.Errorf("==> %s sandbox restore failed: %v\n", name, err)
			_ = r.runPlugins(ctx, "task_end", name, "failed", duration)
			endTrace(statusFailed, duration)
			return taskResult{name: name, duration: duration, status: statusFailed, err: err}
		}
	}

	artifacts, err := r.collectArtifacts(name, task, vars)
	if err != nil {
		r.log.Errorf("==> %s artifacts failed: %v\n", name, err)
		_ = r.runPlugins(ctx, "task_end", name, "failed", duration)
		endTrace(statusFailed, duration)
		return taskResult{name: name, duration: duration, status: statusFailed, err: err}
	}
	if err := r.uploadArtifacts(name, artifacts); err != nil {
		r.log.Errorf("==> %s artifacts upload failed: %v\n", name, err)
		_ = r.runPlugins(ctx, "task_end", name, "failed", duration)
		endTrace(statusFailed, duration)
		return taskResult{name: name, duration: duration, status: statusFailed, err: err}
	}

	r.recordOutputs(name, task, vars)
	r.applyExports(task, vars)

	if err := r.cacheSave(name, task, vars); err != nil {
		r.log.Errorf("==> %s cache save failed: %v\n", name, err)
		_ = r.runPlugins(ctx, "task_end", name, "failed", duration)
		endTrace(statusFailed, duration)
		return taskResult{name: name, duration: duration, status: statusFailed, err: err}
	}

	r.log.Printf("==> %s completed in %s\n", name, formatDuration(duration))
	_ = r.runPlugins(ctx, "task_end", name, "ok", duration)
	endTrace(statusOK, duration)
	return taskResult{name: name, duration: duration, status: statusOK}
}

func (r *Runner) taskVars(plan *taskPlan, name string, task *config.Task) map[string]string {
	vars := map[string]string{}
	for key, value := range r.cfg.Vars {
		vars[key] = value
	}
	for key, value := range r.gitVars {
		if value != "" {
			vars[key] = value
		}
	}
	for key, value := range r.outputsForDeps(plan, name) {
		vars[key] = value
	}
	for key, value := range task.Vars {
		vars[key] = value
	}
	if variant, ok := plan.variants[name]; ok {
		for key, value := range variant.vars {
			vars[key] = value
		}
	}
	return vars
}

func (r *Runner) taskCommands(task *config.Task, vars map[string]string) []string {
	commands := append([]string(nil), task.Run...)
	if task.Docker != nil {
		docker := dockerCommands(task.Docker, vars)
		if len(commands) == 0 {
			commands = docker
		} else {
			commands = append(commands, docker...)
		}
	}
	return commands
}

func (r *Runner) applyExports(task *config.Task, vars map[string]string) {
	if task == nil || len(task.Exports) == 0 {
		return
	}
	r.exportsMu.Lock()
	defer r.exportsMu.Unlock()
	for key, value := range task.Exports {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		r.exports[trimmed] = expandVars(value, vars)
	}
}

func (r *Runner) taskEnv(vars map[string]string, task *config.Task) map[string]string {
	base := envListToMap(r.baseEnv)
	env := map[string]string{}
	for key, value := range base {
		env[key] = value
	}
	for key, value := range r.cfg.Env {
		env[key] = expandVars(value, vars)
	}
	r.exportsMu.Lock()
	for key, value := range r.exports {
		env[key] = expandVars(value, vars)
	}
	r.exportsMu.Unlock()
	for key, value := range task.Env {
		env[key] = expandVars(value, vars)
	}
	return env
}

func (r *Runner) runCommands(ctx context.Context, taskName string, commands []string, opts commandOptions, allowEmpty bool) error {
	if len(commands) == 0 {
		if allowEmpty {
			return nil
		}
		return nil
	}
	return r.runSequential(ctx, taskName, commands, opts)
}

func (r *Runner) runSequential(ctx context.Context, taskName string, commands []string, opts commandOptions) error {
	prefix := ""
	if opts.prefixed {
		prefix = fmt.Sprintf("[%s] ", taskName)
	}
	for _, command := range commands {
		if err := ctx.Err(); err != nil {
			return err
		}
		cmdText := expandVars(command, opts.vars)
		cmdTimeout, cleaned := parseCommandTimeout(cmdText)
		effectiveTimeout := opts.timeout
		if cmdTimeout > 0 {
			effectiveTimeout = cmdTimeout
		}
		if err := r.runCommandWithRetry(ctx, taskName, cleaned, prefix, opts, effectiveTimeout); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) runParallel(ctx context.Context, taskName string, commands []string, opts commandOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(commands))

	for i, command := range commands {
		cmdText := expandVars(command, opts.vars)
		cmdTimeout, cleaned := parseCommandTimeout(cmdText)
		effectiveTimeout := opts.timeout
		if cmdTimeout > 0 {
			effectiveTimeout = cmdTimeout
		}
		prefix := ""
		if opts.prefixed {
			prefix = fmt.Sprintf("[%s:%d] ", taskName, i+1)
		}
		wg.Add(1)
		go func(cmd string, cmdPrefix string, timeout time.Duration) {
			defer wg.Done()
			errCh <- r.runCommandWithRetry(ctx, taskName, cmd, cmdPrefix, opts, timeout)
		}(cleaned, prefix, effectiveTimeout)
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

func (r *Runner) runCommandWithRetry(ctx context.Context, taskName string, command string, prefix string, opts commandOptions, timeout time.Duration) error {
	attempts := opts.retries + 1
	if attempts < 1 {
		attempts = 1
	}
	regexes, err := compileRetryRegex(opts.retryRegex)
	if err != nil {
		return err
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		runCtx := ctx
		cancel := func() {}
		if timeout > 0 {
			runCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		start := time.Now()
		r.trace.CommandStart(taskName, command, attempt)
		capture := newOutputCapture(256 * 1024)
		err := r.runCommand(runCtx, taskName, command, prefix, opts, capture)
		cancel()
		status := "ok"
		if err != nil {
			status = "failed"
		}
		r.trace.CommandEnd(taskName, command, status, time.Since(start), attempt)

		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
		if attempt == attempts {
			return err
		}
		output := capture.String()
		if !shouldRetryCommand(err, output, opts.retryCodes, regexes, opts.retrySignals) {
			return err
		}
		if opts.backoff > 0 {
			delay := opts.backoff * time.Duration(attempt)
			if opts.jitter > 0 {
				delay += r.randomJitter(opts.jitter)
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (r *Runner) runCommand(ctx context.Context, taskName string, command string, prefix string, opts commandOptions, capture *outputCapture) error {
	cmdText := applyLimits(command, opts.limits, opts.shell, opts.remote != nil)
	var cmd *exec.Cmd
	if opts.remote != nil {
		cmd = r.sshCommand(ctx, opts.remote, cmdText, opts.env, opts.workdir, opts.shell)
	} else {
		cmd = platform.ShellCommandWith(ctx, opts.shell, cmdText)
	}
	platform.ConfigureCommand(cmd)
	cmd.Env = opts.env
	cmd.Stdin = os.Stdin
	if opts.workdir != "" && opts.remote == nil {
		cmd.Dir = opts.workdir
	}

	stdout := r.log.commandWriter(prefix, "stdout", opts.secrets)
	stderr := r.log.commandWriter(prefix, "stderr", opts.secrets)
	if opts.silent {
		stdout = io.Discard
		stderr = io.Discard
	}
	if opts.capture != nil {
		if opts.capture.Combined != nil {
			stdout = newMultiWriter(stdout, opts.capture.Combined)
			stderr = newMultiWriter(stderr, opts.capture.Combined)
		}
		if opts.capture.Stdout != nil {
			stdout = newMultiWriter(stdout, opts.capture.Stdout)
		}
		if opts.capture.Stderr != nil {
			stderr = newMultiWriter(stderr, opts.capture.Stderr)
		}
	}
	if capture != nil {
		stdout = newCaptureWriter(stdout, capture)
		stderr = newCaptureWriter(stderr, capture)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		exitCode := 1
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return &commandError{cmd: command, code: exitCode, signal: platform.ExitSignal(err), err: err}
	}

	if flusher, ok := cmd.Stdout.(interface{ Flush() }); ok {
		flusher.Flush()
	}
	if flusher, ok := cmd.Stderr.(interface{ Flush() }); ok {
		flusher.Flush()
	}

	return nil
}

func (r *Runner) printSummary(plan *taskPlan, results map[string]taskResult, total time.Duration) {
	summary := buildSummary(plan, results, total)
	r.log.Printf("==> summary\n")
	for _, name := range plan.order {
		result, ok := results[name]
		if !ok {
			r.log.Printf("%s skipped\n", name)
			continue
		}
		switch result.status {
		case statusFailed:
			r.log.Printf("%s failed in %s\n", name, formatDuration(result.duration))
		case statusUpToDate:
			if r.opts.Explain && result.reason != "" {
				r.log.Printf("%s up-to-date (%s)\n", name, result.reason)
			} else {
				r.log.Printf("%s up-to-date\n", name)
			}
		case statusSkipped, statusCanceled:
			if r.opts.Explain && result.reason != "" {
				r.log.Printf("%s skipped (%s)\n", name, result.reason)
			} else {
				r.log.Printf("%s skipped\n", name)
			}
		default:
			if r.opts.Explain && result.reason != "" {
				r.log.Printf("%s completed in %s (%s)\n", name, formatDuration(result.duration), result.reason)
			} else {
				r.log.Printf("%s completed in %s\n", name, formatDuration(result.duration))
			}
		}
	}

	r.log.Printf("==> total %s (ok=%d failed=%d skipped=%d up-to-date=%d)\n", summary.TotalDuration, summary.Ok, summary.Failed, summary.Skipped, summary.UpToDate)

	if r.opts.Profile {
		r.printProfile(plan, results)
	}
	r.writeSummaryJSON(summary)
}
