package platform

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
)

func ShellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func ShellCommandWith(ctx context.Context, shell string, command string) *exec.Cmd {
	if strings.TrimSpace(shell) == "" {
		return ShellCommand(ctx, command)
	}
	parts := strings.Fields(shell)
	if len(parts) > 1 {
		args := append(parts[1:], command)
		return exec.CommandContext(ctx, parts[0], args...)
	}

	switch parts[0] {
	case "sh", "bash", "zsh", "ksh":
		return exec.CommandContext(ctx, parts[0], "-c", command)
	case "pwsh":
		return exec.CommandContext(ctx, "pwsh", "-NoProfile", "-Command", command)
	case "powershell":
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", command)
	case "cmd":
		return exec.CommandContext(ctx, "cmd", "/C", command)
	default:
		return exec.CommandContext(ctx, parts[0], "-c", command)
	}
}
