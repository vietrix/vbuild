package runner

import (
	"fmt"
	"os"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) recordCheckpoint(taskName string, task *config.Task, vars map[string]string) (string, error) {
	if task == nil || task.Checkpoint == nil {
		return "", nil
	}
	paths, err := r.resolvePatterns(task.Checkpoint.Paths, vars)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("checkpoint: no files matched")
	}
	var latest string
	var latestTime time.Time
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latest = path
		}
	}
	if latest == "" {
		return "", fmt.Errorf("checkpoint: no files found")
	}
	key := task.Checkpoint.Var
	if key != "" {
		r.outputsMu.Lock()
		if r.outputs[taskName] == nil {
			r.outputs[taskName] = map[string]string{}
		}
		r.outputs[taskName][key] = latest
		r.outputsMu.Unlock()
	}
	return latest, nil
}
