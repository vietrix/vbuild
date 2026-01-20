package runner

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) applyArgsVars(taskName string, task *config.Task, vars map[string]string) {
	if task == nil {
		return
	}
	if r.opts.ArgsTarget == "" || r.opts.ArgsTarget != taskName {
		return
	}
	args := r.opts.Args
	shell := task.Shell
	if shell == "" && r.cfg != nil && r.cfg.Defaults != nil {
		shell = r.cfg.Defaults.Shell
	}
	if strings.TrimSpace(shell) == "" {
		if runtime.GOOS == "windows" {
			shell = "powershell"
		} else {
			shell = "sh"
		}
	}
	quoted := ""
	raw := ""
	if len(args) > 0 {
		quoted = joinShellArgs(args, shell)
		raw = strings.Join(args, " ")
	}
	argc := strconv.Itoa(len(args))
	vars["VBUILD_ARGC"] = argc
	vars["VBUILD_ARGS"] = quoted
	vars["VBUILD_ARGS_RAW"] = raw
	if _, ok := vars["ARGC"]; !ok {
		vars["ARGC"] = argc
	}
	if _, ok := vars["ARGS"]; !ok {
		vars["ARGS"] = quoted
	}
	if _, ok := vars["ARGS_RAW"]; !ok {
		vars["ARGS_RAW"] = raw
	}
	for i, arg := range args {
		key := fmt.Sprintf("VBUILD_ARG_%d", i)
		vars[key] = arg
		alias := fmt.Sprintf("ARG_%d", i)
		if _, ok := vars[alias]; !ok {
			vars[alias] = arg
		}
	}
}
