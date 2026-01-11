package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func (r *Runner) OpenShell(taskName string) error {
	if taskName == "" {
		taskName = "default"
	}
	plan, err := r.buildPlan([]string{taskName})
	if err != nil {
		return err
	}
	task, ok := plan.tasks[taskName]
	if !ok || task == nil {
		return fmt.Errorf("unknown task: %s", taskName)
	}
	if task.Remote != nil {
		return fmt.Errorf("shell does not support remote tasks")
	}

	vars := r.taskVars(plan, taskName, task)
	envMap := r.taskEnv(vars, task)
	env := envMapToList(envMap)
	workdir := expandVars(task.Workdir, vars)
	if workdir == "" {
		workdir = r.configRoot
	}
	if !filepath.IsAbs(workdir) {
		workdir = r.resolvePath(workdir)
	}

	shell, args := interactiveShell(task.Shell)
	cmd := exec.Command(shell, args...)
	cmd.Env = env
	cmd.Dir = workdir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func interactiveShell(taskShell string) (string, []string) {
	taskShell = strings.TrimSpace(taskShell)
	if taskShell != "" {
		lower := strings.ToLower(taskShell)
		if strings.Contains(lower, "-c") || strings.Contains(lower, "-command") {
			taskShell = ""
		} else {
			parts := strings.Fields(taskShell)
			if len(parts) > 0 {
				return parts[0], parts[1:]
			}
		}
	}
	if runtime.GOOS == "windows" {
		return "powershell", []string{"-NoProfile"}
	}
	return "sh", []string{}
}
