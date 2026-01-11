package runner

import (
	"encoding/json"
	"io"
	"sort"
)

type listTask struct {
	Name    string   `json:"name"`
	Desc    string   `json:"desc,omitempty"`
	Deps    []string `json:"deps,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
}

func (r *Runner) ListTasksJSON(out io.Writer) error {
	if out == nil {
		out = r.log.out
	}
	names := make([]string, 0, len(r.cfg.Tasks))
	for name := range r.cfg.Tasks {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]listTask, 0, len(names))
	for _, name := range names {
		task := r.cfg.Tasks[name]
		item := listTask{Name: name}
		if task != nil {
			item.Desc = task.Desc
			item.Deps = append([]string(nil), task.Deps...)
			item.Tags = append([]string(nil), task.Tags...)
			item.Aliases = append([]string(nil), task.Aliases...)
		}
		items = append(items, item)
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}
