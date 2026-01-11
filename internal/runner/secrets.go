package runner

import (
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) resolveSecrets(names []string, env []string) []string {
	if len(names) == 0 {
		return []string{}
	}
	values := []string{}
	envMap := envListToMap(env)
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if value, ok := envMap[name]; ok && value != "" {
			values = append(values, value)
		}
	}
	return dedupeNonEmpty(values)
}

func (r *Runner) taskSecrets(task *config.Task, env map[string]string) []string {
	values := append([]string(nil), r.secrets...)
	for _, name := range task.Secrets {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if value, ok := env[name]; ok && value != "" {
			values = append(values, value)
		}
	}
	return dedupeNonEmpty(values)
}

func dedupeNonEmpty(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
