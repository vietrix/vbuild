//go:build !windows

package windows

import "os/exec"

func ConfigureCommand(cmd *exec.Cmd) {
	_ = cmd
}
