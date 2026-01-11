package runner

import (
	"io"
	"os/exec"

	"github.com/vietrix/vbuild/internal/config"
)

type logPlugin struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func (r *Runner) startLogPlugins(plugins []config.Plugin) []logPlugin {
	if len(plugins) == 0 {
		return nil
	}
	running := []logPlugin{}
	for _, plugin := range plugins {
		if plugin.Command == "" {
			continue
		}
		cmd := exec.Command(plugin.Command, plugin.Args...)
		cmd.Env = append([]string(nil), r.baseEnv...)
		cmd.Env = append(cmd.Env, "VBUILD_LOG_PLUGIN=1")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			r.log.Errorf("log plugin %s failed to open stdin: %v\n", plugin.Command, err)
			continue
		}
		cmd.Stdout = r.log.err
		cmd.Stderr = r.log.err
		if err := cmd.Start(); err != nil {
			_ = stdin.Close()
			r.log.Errorf("log plugin %s failed to start: %v\n", plugin.Command, err)
			continue
		}
		r.log.AddHook(stdin)
		running = append(running, logPlugin{cmd: cmd, stdin: stdin})
	}
	return running
}

func (r *Runner) closeLogPlugins() {
	for _, plugin := range r.logPlugins {
		_ = plugin.stdin.Close()
	}
	for _, plugin := range r.logPlugins {
		_ = plugin.cmd.Wait()
	}
}
