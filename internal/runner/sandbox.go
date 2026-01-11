package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

type sandboxContext struct {
	root    string
	workdir string
}

func (s *sandboxContext) cleanup() {
	if s == nil || s.root == "" {
		return
	}
	_ = os.RemoveAll(s.root)
}

func (r *Runner) prepareSandbox(taskName string, task *config.Task, vars map[string]string, workdir string) (*sandboxContext, error) {
	sandboxRoot := filepath.Join(r.configRoot, ".vbuild", "sandbox", fmt.Sprintf("%s-%d", sanitizePath(taskName), time.Now().UnixNano()))
	if err := os.MkdirAll(sandboxRoot, 0o755); err != nil {
		return nil, fmt.Errorf("sandbox create: %w", err)
	}

	inputs, err := r.resolvePatternsWithRoot(task.Inputs, vars, r.configRoot)
	if err != nil {
		return nil, fmt.Errorf("sandbox inputs: %w", err)
	}
	for _, input := range inputs {
		rel, err := filepath.Rel(r.configRoot, input)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("sandbox input outside repo: %s", input)
		}
		dest := filepath.Join(sandboxRoot, rel)
		if err := copyFile(input, dest); err != nil {
			return nil, fmt.Errorf("sandbox copy: %w", err)
		}
	}

	relWorkdir := strings.TrimSpace(workdir)
	if relWorkdir == "" {
		relWorkdir = "."
	}
	if filepath.IsAbs(relWorkdir) {
		rel, err := filepath.Rel(r.configRoot, relWorkdir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("sandbox workdir outside repo: %s", workdir)
		}
		relWorkdir = rel
	}
	sandboxWorkdir := filepath.Join(sandboxRoot, relWorkdir)
	if err := os.MkdirAll(sandboxWorkdir, 0o755); err != nil {
		return nil, fmt.Errorf("sandbox workdir: %w", err)
	}

	return &sandboxContext{root: sandboxRoot, workdir: sandboxWorkdir}, nil
}

func (r *Runner) restoreSandboxOutputs(task *config.Task, vars map[string]string, sandboxRoot string) error {
	outputs, err := r.resolveOutputFilesWithRoot(task, vars, sandboxRoot)
	if err != nil {
		return err
	}
	for _, output := range outputs {
		rel, err := filepath.Rel(sandboxRoot, output)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		dest := filepath.Join(r.configRoot, rel)
		if err := copyFile(output, dest); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) resolveOutputFilesWithRoot(task *config.Task, vars map[string]string, root string) ([]string, error) {
	patterns := append([]string{}, task.Outputs...)
	patterns = append(patterns, task.OutputPaths...)
	return r.resolvePatternsWithRoot(dedupeNonEmpty(patterns), vars, root)
}
