package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

func (r *Runner) runPlugins(ctx context.Context, event, task, status string, duration time.Duration) error {
	if r.cfg == nil || len(r.cfg.Plugins) == 0 {
		return nil
	}
	for _, plugin := range r.cfg.Plugins {
		if plugin.Command == "" {
			continue
		}
		args := append([]string{}, plugin.Args...)
		cmd := exec.CommandContext(ctx, plugin.Command, args...)
		cmd.Env = append(r.baseEnv, []string{
			"VBUILD_EVENT=" + event,
			"VBUILD_TASK=" + task,
			"VBUILD_STATUS=" + status,
			"VBUILD_DURATION_MS=" + strconv.FormatInt(duration.Milliseconds(), 10),
		}...)
		cmd.Stdout = r.log.out
		cmd.Stderr = r.log.err
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("plugin %s failed: %w", plugin.Command, err)
		}
	}
	return nil
}
