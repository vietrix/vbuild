package runner

import "github.com/vietrix/vbuild/internal/config"

func (r *Runner) applyOffline(env map[string]string, task *config.Task) {
	enabled := false
	if r.cfg != nil && r.cfg.Offline != nil && r.cfg.Offline.Enabled {
		enabled = true
	}
	if task != nil && task.Offline != nil && task.Offline.Enabled {
		enabled = true
	}
	if !enabled {
		return
	}
	env["VBUILD_OFFLINE"] = "1"
	env["HF_HUB_OFFLINE"] = "1"
	env["HF_DATASETS_OFFLINE"] = "1"
	env["TRANSFORMERS_OFFLINE"] = "1"
	env["WANDB_MODE"] = "offline"
	if r.cfg != nil && r.cfg.Offline != nil {
		for key, value := range r.cfg.Offline.Env {
			env[key] = value
		}
	}
	if task != nil && task.Offline != nil {
		for key, value := range task.Offline.Env {
			env[key] = value
		}
	}
}
