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
	variants     map[string]taskVariant
}

type taskVariant struct {
	base string
	vars map[string]string
}

func (r *Runner) buildPlan(targets []string) (*taskPlan, error) {
	visited := map[string]bool{}
	visiting := map[string]bool{}
	tasks := map[string]*config.Task{}
	deps := map[string][]string{}

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
		}
		for _, dep := range task.DependsOn {
			depName := strings.TrimSpace(dep.Task)
			if depName == "" {
				return fmt.Errorf("task %s has empty depends_on task", name)
			}
			if err := visit(depName); err != nil {
				return err
			}
		}
		visiting[name] = false
		visited[name] = true
		tasks[name] = cloneTask(task)
		deps[name] = append([]string(nil), task.Deps...)
		return nil
	}

	for _, target := range targets {
		if strings.TrimSpace(target) == "" {
			return nil, fmt.Errorf("task name is required")
		}
		if err := visit(target); err != nil {
			return nil, err
		}
	}

	variants := map[string]taskVariant{}
	if err := expandMatrix(tasks, deps, variants); err != nil {
		return nil, err
	}
	if err := r.applyConditionalDeps(tasks, deps, variants); err != nil {
		return nil, err
	}
	if err := expandFanout(tasks, deps, variants); err != nil {
		return nil, err
	}
	pruneTasks(targets, tasks, deps, variants)
	propagateOutputPaths(tasks, deps)

	dependents := map[string][]string{}
	for name := range tasks {
		if _, ok := deps[name]; !ok {
			deps[name] = []string{}
		}
	}
	for name, depList := range deps {
		for _, dep := range depList {
			dependents[dep] = append(dependents[dep], name)
		}
	}
	for name := range dependents {
		sort.Strings(dependents[name])
	}

	order, err := topoSort(tasks, deps)
	if err != nil {
		return nil, err
	}

	return &taskPlan{
		tasks:        tasks,
		deps:         deps,
		dependents:   dependents,
		order:        order,
		prefixOutput: len(tasks) > 1,
		variants:     variants,
	}, nil
}

func topoSort(tasks map[string]*config.Task, deps map[string][]string) ([]string, error) {
	inDegree := map[string]int{}
	for name := range tasks {
		inDegree[name] = 0
	}
	for name, depList := range deps {
		if _, ok := tasks[name]; !ok {
			continue
		}
		inDegree[name] += len(depList)
	}

	ready := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(tasks))
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		order = append(order, name)

		for _, dependent := range depsDependents(name, deps) {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				ready = append(ready, dependent)
				sort.Strings(ready)
			}
		}
	}

	if len(order) != len(tasks) {
		return nil, fmt.Errorf("dependency cycle detected")
	}
	return order, nil
}

func depsDependents(name string, deps map[string][]string) []string {
	out := []string{}
	for task, depList := range deps {
		for _, dep := range depList {
			if dep == name {
				out = append(out, task)
				break
			}
		}
	}
	return out
}

func expandMatrix(tasks map[string]*config.Task, deps map[string][]string, variants map[string]taskVariant) error {
	for name, task := range tasks {
		if task == nil || len(task.Matrix) == 0 {
			continue
		}
		combos := matrixCombos(task.Matrix)
		if len(combos) == 0 {
			continue
		}
		origDeps := append([]string(nil), deps[name]...)
		variantNames := make([]string, 0, len(combos))
		for _, combo := range combos {
			variantName := formatVariantName(name, combo)
			if _, exists := tasks[variantName]; exists {
				return fmt.Errorf("matrix variant name collision: %s", variantName)
			}
			variant := cloneTask(task)
			variant.Matrix = nil
			if variant.Vars == nil {
				variant.Vars = map[string]string{}
			}
			if variant.Env == nil {
				variant.Env = map[string]string{}
			}
			for key, value := range combo {
				variant.Vars[key] = value
				if _, ok := variant.Env[key]; !ok {
					variant.Env[key] = value
				}
			}
			tasks[variantName] = variant
			deps[variantName] = append([]string(nil), origDeps...)
			variants[variantName] = taskVariant{base: name, vars: combo}
			variantNames = append(variantNames, variantName)
		}
		agg := cloneTask(task)
		agg.Run = nil
		agg.Pre = nil
		agg.Post = nil
		agg.Matrix = nil
		agg.Deps = append([]string(nil), variantNames...)
		tasks[name] = agg
		deps[name] = append([]string(nil), variantNames...)
	}
	return nil
}

func expandFanout(tasks map[string]*config.Task, deps map[string][]string, variants map[string]taskVariant) error {
	for name, task := range tasks {
		if task == nil || !task.Fanout || len(task.Run) == 0 {
			continue
		}
		baseDeps := append([]string(nil), deps[name]...)
		fanoutDeps := []string{}
		var preName string
		if len(task.Pre) > 0 {
			preName = fanoutTaskName(name, "pre")
			if _, exists := tasks[preName]; exists {
				return fmt.Errorf("fanout task name collision: %s", preName)
			}
			preTask := cloneTask(task)
			preTask.Run = append([]string(nil), task.Pre...)
			preTask.Pre = nil
			preTask.Post = nil
			preTask.Fanout = false
			preTask.Parallel = false
			tasks[preName] = preTask
			deps[preName] = append([]string(nil), baseDeps...)
			if variant, ok := variants[name]; ok {
				variants[preName] = variant
			}
		}

		for i, cmd := range task.Run {
			childName := fanoutTaskName(name, fmt.Sprintf("%d", i+1))
			if _, exists := tasks[childName]; exists {
				return fmt.Errorf("fanout task name collision: %s", childName)
			}
			child := cloneTask(task)
			child.Run = []string{cmd}
			child.Pre = nil
			child.Post = nil
			child.Fanout = false
			child.Parallel = false
			tasks[childName] = child
			if preName != "" {
				deps[childName] = []string{preName}
			} else {
				deps[childName] = append([]string(nil), baseDeps...)
			}
			if variant, ok := variants[name]; ok {
				variants[childName] = variant
			}
			fanoutDeps = append(fanoutDeps, childName)
		}

		var postName string
		if len(task.Post) > 0 {
			postName = fanoutTaskName(name, "post")
			if _, exists := tasks[postName]; exists {
				return fmt.Errorf("fanout task name collision: %s", postName)
			}
			postTask := cloneTask(task)
			postTask.Run = append([]string(nil), task.Post...)
			postTask.Pre = nil
			postTask.Post = nil
			postTask.Fanout = false
			postTask.Parallel = false
			tasks[postName] = postTask
			deps[postName] = append([]string(nil), fanoutDeps...)
			if variant, ok := variants[name]; ok {
				variants[postName] = variant
			}
		}

		agg := cloneTask(task)
		agg.Run = nil
		agg.Pre = nil
		agg.Post = nil
		agg.Fanout = false
		agg.Parallel = false
		tasks[name] = agg
		if postName != "" {
			deps[name] = []string{postName}
		} else {
			deps[name] = append([]string(nil), fanoutDeps...)
		}
	}
	return nil
}

func fanoutTaskName(base, suffix string) string {
	return fmt.Sprintf("%s::%s", base, suffix)
}

func (r *Runner) applyConditionalDeps(tasks map[string]*config.Task, deps map[string][]string, variants map[string]taskVariant) error {
	plan := &taskPlan{tasks: tasks, deps: deps, variants: variants}
	for name, task := range tasks {
		if task == nil || len(task.DependsOn) == 0 {
			continue
		}
		vars := r.taskVars(plan, name, task)
		envMap := r.taskEnv(vars, task)
		for _, dep := range task.DependsOn {
			depName := strings.TrimSpace(dep.Task)
			if depName == "" {
				continue
			}
			ok, err := evalCondition(dep.When, vars, envMap)
			if err != nil {
				return err
			}
			if ok {
				deps[name] = append(deps[name], depName)
			}
		}
	}
	for name, depList := range deps {
		deps[name] = dedupeNonEmpty(depList)
		sort.Strings(deps[name])
	}
	return nil
}

func pruneTasks(targets []string, tasks map[string]*config.Task, deps map[string][]string, variants map[string]taskVariant) {
	reachable := map[string]bool{}
	var walk func(string)
	walk = func(name string) {
		if reachable[name] {
			return
		}
		reachable[name] = true
		for _, dep := range deps[name] {
			walk(dep)
		}
	}
	for _, target := range targets {
		walk(target)
	}

	for name := range tasks {
		if !reachable[name] {
			delete(tasks, name)
			delete(deps, name)
			delete(variants, name)
		}
	}
	for name, depList := range deps {
		filtered := []string{}
		for _, dep := range depList {
			if reachable[dep] {
				filtered = append(filtered, dep)
			}
		}
		deps[name] = filtered
	}
}

func propagateOutputPaths(tasks map[string]*config.Task, deps map[string][]string) {
	for name, task := range tasks {
		if task == nil {
			continue
		}
		additional := []string{}
		for _, dep := range deps[name] {
			depTask := tasks[dep]
			if depTask == nil {
				continue
			}
			if len(depTask.OutputPaths) > 0 {
				additional = append(additional, depTask.OutputPaths...)
			}
		}
		if len(additional) == 0 {
			continue
		}
		task.Inputs = append(task.Inputs, additional...)
		task.Inputs = dedupeNonEmpty(task.Inputs)
		sort.Strings(task.Inputs)
	}
}

func matrixCombos(matrix map[string][]string) []map[string]string {
	keys := make([]string, 0, len(matrix))
	for key := range matrix {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var combos []map[string]string
	var walk func(int, map[string]string)
	walk = func(idx int, current map[string]string) {
		if idx == len(keys) {
			clone := map[string]string{}
			for k, v := range current {
				clone[k] = v
			}
			combos = append(combos, clone)
			return
		}
		key := keys[idx]
		values := matrix[key]
		for _, value := range values {
			next := map[string]string{}
			for k, v := range current {
				next[k] = v
			}
			next[key] = value
			walk(idx+1, next)
		}
	}
	walk(0, map[string]string{})
	return combos
}

func formatVariantName(base string, vars map[string]string) string {
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := sanitizeVariantValue(vars[key])
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return fmt.Sprintf("%s[%s]", base, strings.Join(parts, ","))
}

func sanitizeVariantValue(value string) string {
	out := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			out.WriteRune(r)
		} else {
			out.WriteRune('_')
		}
	}
	if out.Len() == 0 {
		return "value"
	}
	return out.String()
}
