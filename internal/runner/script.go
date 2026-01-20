package runner

import (
	"runtime"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) buildScriptCommand(taskName string, task *config.Task) (string, []string) {
	if task == nil {
		return "", nil
	}
	script := strings.TrimSpace(task.Script)
	if script == "" {
		return "", nil
	}
	shell := resolveTaskShell(r.cfg, task)
	args := []string{}
	if r.opts.ArgsTarget == taskName {
		args = r.opts.Args
	}
	placeholders := parseScriptPlaceholders(script)
	placeholderSet := map[string]struct{}{}
	for _, name := range placeholders {
		placeholderSet[normalizePlaceholderName(name)] = struct{}{}
	}
	values, leftovers := parseArgsForPlaceholders(args, placeholderSet)
	cmd, missing := replaceScriptPlaceholders(script, values, shell)
	if len(args) > 0 && !scriptUsesArgsVars(script) && len(leftovers) > 0 {
		cmd = cmd + " " + joinShellArgs(leftovers, shell)
	}
	return cmd, missing
}

func resolveTaskShell(cfg *config.Config, task *config.Task) string {
	shell := ""
	if task != nil {
		shell = task.Shell
	}
	if shell == "" && cfg != nil && cfg.Defaults != nil {
		shell = cfg.Defaults.Shell
	}
	if strings.TrimSpace(shell) == "" {
		if runtime.GOOS == "windows" {
			return "powershell"
		}
		return "sh"
	}
	return shell
}

func scriptUsesArgsVars(script string) bool {
	return strings.Contains(script, "{{ARGS") ||
		strings.Contains(script, "{{ARGC") ||
		strings.Contains(script, "{{ARG_") ||
		strings.Contains(script, "{{VBUILD_ARGS") ||
		strings.Contains(script, "{{VBUILD_ARGC") ||
		strings.Contains(script, "{{VBUILD_ARG_")
}

func parseScriptPlaceholders(script string) []string {
	seen := map[string]struct{}{}
	placeholders := []string{}
	for i := 0; i < len(script); {
		if strings.HasPrefix(script[i:], "{{") {
			end := strings.Index(script[i+2:], "}}")
			if end >= 0 {
				i += end + 4
				continue
			}
		}
		if script[i] == '{' {
			end := strings.IndexByte(script[i+1:], '}')
			if end >= 0 {
				name := script[i+1 : i+1+end]
				if isScriptPlaceholder(name) {
					if _, ok := seen[name]; !ok {
						seen[name] = struct{}{}
						placeholders = append(placeholders, name)
					}
					i += end + 2
					continue
				}
			}
		}
		i++
	}
	return placeholders
}

func replaceScriptPlaceholders(script string, values map[string]string, shell string) (string, []string) {
	var builder strings.Builder
	missing := []string{}
	for i := 0; i < len(script); {
		if strings.HasPrefix(script[i:], "{{") {
			end := strings.Index(script[i+2:], "}}")
			if end >= 0 {
				builder.WriteString(script[i : i+end+4])
				i += end + 4
				continue
			}
		}
		if script[i] == '{' {
			end := strings.IndexByte(script[i+1:], '}')
			if end >= 0 {
				name := script[i+1 : i+1+end]
				if isScriptPlaceholder(name) {
					normalized := normalizePlaceholderName(name)
					if value, ok := values[normalized]; ok {
						builder.WriteString(quoteScriptValue(value, shell))
					} else {
						missing = append(missing, normalized)
						builder.WriteByte('{')
						builder.WriteString(name)
						builder.WriteByte('}')
					}
					i += end + 2
					continue
				}
			}
		}
		builder.WriteByte(script[i])
		i++
	}
	return builder.String(), missing
}

func parseArgsForPlaceholders(args []string, placeholders map[string]struct{}) (map[string]string, []string) {
	values := map[string]string{}
	if len(args) == 0 {
		return values, nil
	}
	if len(placeholders) == 0 {
		return values, append([]string(nil), args...)
	}
	leftovers := []string{}
	for i := 0; i < len(args); {
		arg := args[i]
		if arg == "--" {
			leftovers = append(leftovers, args[i:]...)
			break
		}
		if strings.HasPrefix(arg, "--") && len(arg) > 2 {
			name, value, hasValue, consumed := splitPlaceholderFlag(args, i, placeholders)
			if name != "" {
				if hasValue {
					values[normalizePlaceholderName(name)] = value
				}
				i += consumed
				continue
			}
		}
		leftovers = append(leftovers, arg)
		i++
	}
	return values, leftovers
}

func splitPlaceholderFlag(args []string, index int, placeholders map[string]struct{}) (string, string, bool, int) {
	raw := strings.TrimPrefix(args[index], "--")
	if raw == "" {
		return "", "", false, 1
	}
	if eq := strings.Index(raw, "="); eq >= 0 {
		name := raw[:eq]
		if _, ok := placeholders[normalizePlaceholderName(name)]; !ok {
			return "", "", false, 1
		}
		return name, raw[eq+1:], true, 1
	}
	name := raw
	if _, ok := placeholders[normalizePlaceholderName(name)]; !ok {
		return "", "", false, 1
	}
	if index+1 >= len(args) || args[index+1] == "--" {
		return name, "", false, 1
	}
	return name, args[index+1], true, 2
}

func normalizePlaceholderName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	return strings.ToLower(name)
}

func isScriptPlaceholder(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
	}
	return true
}

func quoteScriptValue(value string, shell string) string {
	if shellKind(shell) == "powershell" {
		return powershellQuote(value)
	}
	return shellQuote(value)
}

func formatMissingArgs(taskName string, missing []string) string {
	missing = dedupeNonEmpty(missing)
	formatted := make([]string, 0, len(missing))
	for _, name := range missing {
		formatted = append(formatted, "--"+name)
	}
	if len(formatted) == 0 {
		return ""
	}
	return "task " + taskName + " missing required args: " + strings.Join(formatted, ", ")
}
