package runner

import (
	"sort"
	"strings"
	"time"
)

func (r *Runner) printProfile(plan *taskPlan, results map[string]taskResult) {
	durations := map[string]time.Duration{}
	for name := range plan.tasks {
		if result, ok := results[name]; ok {
			durations[name] = result.duration
		} else {
			durations[name] = 0
		}
	}

	dist := map[string]time.Duration{}
	prev := map[string]string{}
	for _, name := range plan.order {
		maxDep := time.Duration(0)
		prevTask := ""
		for _, dep := range plan.deps[name] {
			if dist[dep] > maxDep {
				maxDep = dist[dep]
				prevTask = dep
			}
		}
		dist[name] = maxDep + durations[name]
		if prevTask != "" {
			prev[name] = prevTask
		}
	}

	maxTask := ""
	maxDuration := time.Duration(0)
	for name, duration := range dist {
		if duration > maxDuration {
			maxDuration = duration
			maxTask = name
		}
	}

	path := []string{}
	for current := maxTask; current != ""; current = prev[current] {
		path = append([]string{current}, path...)
	}

	r.log.Printf("==> profile\n")
	if len(path) > 0 {
		r.log.Printf("critical path %s: %s\n", formatDuration(maxDuration), strings.Join(path, " -> "))
	}

	names := make([]string, 0, len(durations))
	for name := range durations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		r.log.Printf("%-24s %s\n", name, formatDuration(durations[name]))
	}
}
