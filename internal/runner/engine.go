package runner

import (
	"fmt"
	"io"
	"sort"

	"github.com/vietrix/vbuild/internal/config"
)

type Runner struct {
	cfg    *config.Config
	dryRun bool
	log    *logger
}

func New(cfg *config.Config, dryRun bool, stdout, stderr io.Writer) *Runner {
	return &Runner{cfg: cfg, dryRun: dryRun, log: newLogger(stdout, stderr)}
}

func (r *Runner) ListTasks() {
	names := make([]string, 0, len(r.cfg.Tasks))
	for name := range r.cfg.Tasks {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		desc := ""
		if r.cfg.Tasks[name] != nil {
			desc = r.cfg.Tasks[name].Desc
		}
		if desc == "" {
			r.log.Printf("%s\n", name)
			continue
		}
		r.log.Printf("%-16s %s\n", name, desc)
	}
}

func (r *Runner) Run(taskName string) error {
	if taskName == "" {
		return fmt.Errorf("task name is required")
	}

	plan, err := r.buildPlan(taskName)
	if err != nil {
		return err
	}

	if r.dryRun {
		r.printDryRun(plan)
		return nil
	}

	return r.executePlan(plan)
}
