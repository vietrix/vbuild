//go:build !windows

package unix

import (
	"os/exec"
	"syscall"
)

func ConfigureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
