package runner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

type taskPlan struct {
	tasks        map[string]*config.Task
	deps         map[string][]string
	dependents   map[string][]string
	order        []string
	prefixOutput bool
}

func (r *Runner) buildPlan(taskName string) (*taskPlan, error) {
	visited := map[string]bool{}
	visiting := map[string]bool{}
	order := []string{}
	tasks := map[string]*config.Task{}
	deps := map[string][]string{}
	dependents := map[string][]string{}

	var visit func(name string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("dependency cycle detected at %s", name)
		}
		if visited[name] {
			return nil
		}

		task, ok := r.cfg.Tasks[name]
		if !ok || task == nil {
			return fmt.Errorf("unknown task: %s", name)
		}

		visiting[name] = true
		for _, dep := range task.Deps {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				return fmt.Errorf("task %s has empty dependency", name)
			}
			if err := visit(dep); err != nil {
				return err
			}
			dependents[dep] = append(dependents[dep], name)
		}
		visiting[name] = false
		visited[name] = true
		order = append(order, name)
		tasks[name] = task
		deps[name] = append([]string(nil), task.Deps...)
		return nil
	}

	if err := visit(taskName); err != nil {
		return nil, err
	}

	for name := range dependents {
		sort.Strings(dependents[name])
	}

	return &taskPlan{
		tasks:        tasks,
		deps:         deps,
		dependents:   dependents,
		order:        order,
		prefixOutput: len(tasks) > 1,
	}, nil
}
