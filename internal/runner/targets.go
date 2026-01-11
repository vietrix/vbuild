package runner

import (
	"fmt"
	"sort"
	"strings"
)

func (r *Runner) resolveTargets(targets []string) ([]string, error) {
	if r.cfg == nil || len(targets) == 0 {
		return targets, nil
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, name := range targets {
		resolved, err := r.resolveTarget(name)
		if err != nil {
			return nil, err
		}
		for _, item := range resolved {
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *Runner) resolveTarget(name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("task name is required")
	}
	if r.cfg.Aliases != nil {
		if actual, ok := r.cfg.Aliases[name]; ok {
			name = actual
		}
	}
	if _, ok := r.cfg.Tasks[name]; ok {
		return []string{name}, nil
	}
	group := strings.TrimSuffix(name, ":")
	prefix := group + ":"
	groupMatches := []string{}
	for taskName := range r.cfg.Tasks {
		if strings.HasPrefix(taskName, prefix) {
			groupMatches = append(groupMatches, taskName)
		}
	}
	if len(groupMatches) > 0 {
		sort.Strings(groupMatches)
		return groupMatches, nil
	}
	return nil, fmt.Errorf("unknown task: %s", name)
}
