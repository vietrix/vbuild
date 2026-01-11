package runner

import "sort"

func (r *Runner) TasksByTag(tag string) []string {
	if tag == "" {
		return nil
	}
	names := []string{}
	for name, task := range r.cfg.Tasks {
		if task == nil {
			continue
		}
		for _, t := range task.Tags {
			if t == tag {
				names = append(names, name)
				break
			}
		}
	}
	sort.Strings(names)
	return names
}
