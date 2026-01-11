package runner

import (
	"fmt"
	"os"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

type fileSnapshot struct {
	ModTime int64
	Size    int64
}

func (r *Runner) Watch(taskName string, interval, debounce time.Duration) error {
	if taskName == "" {
		return fmt.Errorf("task name is required")
	}
	task, ok := r.cfg.Tasks[taskName]
	if !ok || task == nil {
		return fmt.Errorf("unknown task: %s", taskName)
	}

	vars := r.taskVars(&taskPlan{variants: map[string]taskVariant{}}, taskName, task)
	patterns := watchPatterns(task)
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	snapshot, err := r.buildSnapshot(patterns, vars)
	if err != nil {
		return err
	}

	r.log.Printf("==> watching %s\n", taskName)
	for {
		time.Sleep(interval)
		next, err := r.buildSnapshot(patterns, vars)
		if err != nil {
			r.log.Errorf("watch error: %v\n", err)
			continue
		}
		if !snapshotChanged(snapshot, next) {
			continue
		}
		if debounce > 0 {
			for {
				time.Sleep(debounce)
				latest, err := r.buildSnapshot(patterns, vars)
				if err != nil {
					r.log.Errorf("watch error: %v\n", err)
					break
				}
				if !snapshotChanged(next, latest) {
					next = latest
					break
				}
				next = latest
			}
		}
		snapshot = next
		_ = r.RunTargets([]string{taskName})
	}
}

func watchPatterns(task *config.Task) []string {
	if len(task.Watch) > 0 {
		return task.Watch
	}
	if len(task.Inputs) > 0 {
		return task.Inputs
	}
	return nil
}

func (r *Runner) buildSnapshot(patterns []string, vars map[string]string) (map[string]fileSnapshot, error) {
	files, err := r.resolvePatterns(patterns, vars)
	if err != nil {
		return nil, err
	}
	snapshot := map[string]fileSnapshot{}
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		snapshot[path] = fileSnapshot{ModTime: info.ModTime().UnixNano(), Size: info.Size()}
	}
	return snapshot, nil
}

func snapshotChanged(a, b map[string]fileSnapshot) bool {
	if len(a) != len(b) {
		return true
	}
	for path, info := range a {
		other, ok := b[path]
		if !ok {
			return true
		}
		if info.ModTime != other.ModTime || info.Size != other.Size {
			return true
		}
	}
	return false
}
