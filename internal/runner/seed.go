package runner

import (
	"strconv"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) resolveSeed(task *config.Task) int64 {
	if task != nil && task.Seed != 0 {
		return task.Seed
	}
	if r.cfg != nil && r.cfg.Seed != 0 {
		return r.cfg.Seed
	}
	return 0
}

func (r *Runner) applySeed(vars map[string]string, env map[string]string, task *config.Task) int64 {
	seed := r.resolveSeed(task)
	if seed == 0 {
		return 0
	}
	seedText := strconv.FormatInt(seed, 10)
	vars["SEED"] = seedText
	vars["VBUILD_SEED"] = seedText
	env["VBUILD_SEED"] = seedText
	if r.cfg != nil {
		for key, value := range r.cfg.SeedEnv {
			env[key] = expandVars(value, vars)
		}
	}
	if task != nil {
		for key, value := range task.SeedEnv {
			env[key] = expandVars(value, vars)
		}
	}
	return seed
}
