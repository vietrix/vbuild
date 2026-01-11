package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

type snapshotPayload struct {
	Task      string            `json:"task"`
	Timestamp string            `json:"timestamp"`
	Env       map[string]string `json:"env,omitempty"`
	Vars      map[string]string `json:"vars,omitempty"`
	Git       map[string]string `json:"git,omitempty"`
	System    map[string]string `json:"system,omitempty"`
}

func (r *Runner) writeSnapshot(taskName string, task *config.Task, vars, env map[string]string) error {
	spec := r.snapshotSpec(task)
	if spec == nil || !spec.Enabled {
		return nil
	}
	path := spec.Path
	if path == "" {
		path = filepath.Join(".vbuild", "snapshots", sanitizePath(taskName)+"-"+time.Now().UTC().Format("20060102T150405Z")+".json")
	}
	path = r.resolvePath(expandVars(path, vars))
	payload := snapshotPayload{
		Task:      taskName,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if spec.Env {
		payload.Env = cloneStringMap(env)
	}
	if spec.Vars {
		payload.Vars = cloneStringMap(vars)
	}
	if spec.Git {
		payload.Git = cloneStringMap(r.gitVars)
	}
	if spec.System {
		hostname, _ := os.Hostname()
		payload.System = map[string]string{
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
			"cpu":      itoa(runtime.NumCPU()),
			"hostname": hostname,
			"go":       runtime.Version(),
		}
	}
	return writeJSONFile(path, payload)
}

func (r *Runner) snapshotSpec(task *config.Task) *config.SnapshotSpec {
	if task != nil && task.Snapshot != nil {
		return task.Snapshot
	}
	if r.cfg != nil {
		return r.cfg.Snapshot
	}
	return nil
}

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
}
