package runner

import (
	"fmt"
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
	shell := resolveTaskShell(r.cfg, task)
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
	for key, value := range parseNamedArgs(args) {
		if key == "" {
			continue
		}
		normalized := "VBUILD_ARG_" + key
		vars[normalized] = value
		if _, ok := vars[key]; !ok {
			vars[key] = value
		}
	}
}

func parseNamedArgs(args []string) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		name, value, hasValue, consumed := splitNamedArg(args, i)
		if name == "" || !hasValue {
			continue
		}
		key := normalizeFlagKey(name)
		if key == "" {
			continue
		}
		out[key] = value
		if consumed > 1 {
			i += consumed - 1
		}
	}
	return out
}

func splitNamedArg(args []string, index int) (string, string, bool, int) {
	raw := strings.TrimPrefix(args[index], "--")
	if raw == "" || raw == args[index] {
		return "", "", false, 1
	}
	if eq := strings.Index(raw, "="); eq >= 0 {
		return raw[:eq], raw[eq+1:], true, 1
	}
	if index+1 >= len(args) || strings.HasPrefix(args[index+1], "-") {
		return raw, "", false, 1
	}
	return raw, args[index+1], true, 2
}

func normalizeFlagKey(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ToUpper(name)
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return ""
		}
	}
	return name
}
