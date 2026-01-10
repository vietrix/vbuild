package platform

import (
	"os/exec"
	"runtime"

	platformunix "github.com/vietrix/vbuild/internal/platform/unix"
	platformwindows "github.com/vietrix/vbuild/internal/platform/windows"
)

func ConfigureCommand(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		platformwindows.ConfigureCommand(cmd)
		return
	}
	platformunix.ConfigureCommand(cmd)
}
