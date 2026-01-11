package runner

import "github.com/vietrix/vbuild/internal/config"

func effectiveRetries(task *config.Task, cfg *config.Config) int {
	if task == nil {
		return 0
	}
	retries := task.Retries
	if retries == 0 && cfg != nil && cfg.Defaults != nil {
		retries = cfg.Defaults.Retries
	}
	if retries < 0 {
		retries = 0
	}
	maxRetries := task.MaxRetries
	if maxRetries == 0 && cfg != nil && cfg.Defaults != nil {
		maxRetries = cfg.Defaults.MaxRetries
	}
	if maxRetries > 0 && retries > maxRetries {
		return maxRetries
	}
	return retries
}
