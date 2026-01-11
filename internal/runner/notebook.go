package runner

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) runNotebook(ctx context.Context, taskName string, task *config.Task, vars map[string]string, opts commandOptions, prefix string) error {
	if task == nil || task.Notebook == nil {
		return nil
	}
	cmd, err := buildNotebookCommand(task.Notebook, vars, opts.shell)
	if err != nil {
		return err
	}
	return r.runCommandWithRetry(ctx, taskName, cmd, prefix, opts, opts.timeout)
}

func buildNotebookCommand(spec *config.NotebookSpec, vars map[string]string, shell string) (string, error) {
	input := expandVars(spec.Path, vars)
	if input == "" {
		return "", fmt.Errorf("notebook path is empty")
	}
	output := expandVars(spec.Output, vars)
	if output == "" {
		base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
		output = base + ".executed.ipynb"
	}
	if len(spec.Parameters) > 0 {
		if _, err := exec.LookPath("papermill"); err != nil {
			return "", fmt.Errorf("papermill not found (required for notebook parameters)")
		}
		args := []string{"papermill", input, output}
		for key, value := range spec.Parameters {
			args = append(args, "-p", key, value)
		}
		return joinShellArgs(args, shell), nil
	}
	if _, err := exec.LookPath("jupyter"); err == nil {
		args := []string{"jupyter", "nbconvert", "--to", "notebook", "--execute", input, "--output", output}
		return joinShellArgs(args, shell), nil
	}
	if _, err := exec.LookPath("python"); err == nil {
		args := []string{"python", "-m", "jupyter", "nbconvert", "--to", "notebook", "--execute", input, "--output", output}
		return joinShellArgs(args, shell), nil
	}
	return "", fmt.Errorf("jupyter not found (required to execute notebook)")
}

func joinShellArgs(args []string, shell string) string {
	kind := shellKind(shell)
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if kind == "powershell" {
			parts = append(parts, powershellQuote(arg))
		} else {
			parts = append(parts, shellQuote(arg))
		}
	}
	return strings.Join(parts, " ")
}
