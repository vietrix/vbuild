//go:build !windows

package platform

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"
)

func ExitSignal(err error) string {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return ""
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return ""
	}
	return normalizeSignalName(status.Signal())
}

func normalizeSignalName(sig syscall.Signal) string {
	switch sig {
	case syscall.SIGKILL:
		return "SIGKILL"
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGINT:
		return "SIGINT"
	case syscall.SIGQUIT:
		return "SIGQUIT"
	case syscall.SIGABRT:
		return "SIGABRT"
	default:
		name := strings.ToUpper(sig.String())
		if !strings.HasPrefix(name, "SIG") {
			name = "SIG" + name
		}
		return name
	}
}
