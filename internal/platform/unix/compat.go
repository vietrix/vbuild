//go:build windows

package unix

import "os/exec"

func ConfigureCommand(cmd *exec.Cmd) {
	_ = cmd
}
