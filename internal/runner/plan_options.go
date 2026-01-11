package runner

import (
	"fmt"
	"sort"
)

func (p *taskPlan) PrefixUntil(name string) error {
	if p == nil || name == "" {
		return nil
	}
	idx := -1
	for i, item := range p.order {
		if item == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("until task not found: %s", name)
	}
	keep := map[string]bool{}
	for i := 0; i <= idx && i < len(p.order); i++ {
		keep[p.order[i]] = true
	}
	filterPlan(p, keep)
	return nil
}

func (p *taskPlan) Reverse() error {
	if p == nil {
		return nil
	}
	reversed := map[string][]string{}
	for name := range p.tasks {
		reversed[name] = append([]string(nil), p.dependents[name]...)
		sort.Strings(reversed[name])
	}
	dependents := map[string][]string{}
	for name, deps := range reversed {
		for _, dep := range deps {
			dependents[dep] = append(dependents[dep], name)
		}
	}
	for name := range dependents {
		sort.Strings(dependents[name])
	}
	order, err := topoSort(p.tasks, reversed)
	if err != nil {
		return err
	}
	p.deps = reversed
	p.dependents = dependents
	p.order = order
	p.prefixOutput = len(p.tasks) > 1
	return nil
}

func filterPlan(plan *taskPlan, keep map[string]bool) {
	for name := range plan.tasks {
		if !keep[name] {
			delete(plan.tasks, name)
			delete(plan.deps, name)
			delete(plan.dependents, name)
			delete(plan.variants, name)
		}
	}
	for name, depList := range plan.deps {
		filtered := []string{}
		for _, dep := range depList {
			if keep[dep] {
				filtered = append(filtered, dep)
			}
		}
		plan.deps[name] = filtered
	}
	plan.dependents = map[string][]string{}
	for name, depList := range plan.deps {
		for _, dep := range depList {
			plan.dependents[dep] = append(plan.dependents[dep], name)
		}
	}
	for name := range plan.dependents {
		sort.Strings(plan.dependents[name])
	}
	order, _ := topoSort(plan.tasks, plan.deps)
	plan.order = order
	plan.prefixOutput = len(plan.tasks) > 1
}
