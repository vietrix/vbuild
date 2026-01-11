package runner

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) sshCommand(ctx context.Context, remote *config.RemoteSpec, command string, env []string, workdir string, shell string) *exec.Cmd {
	args := []string{}
	if remote.Port != 0 {
		args = append(args, "-p", strconv.Itoa(remote.Port))
	}
	if remote.Identity != "" {
		identity := remote.Identity
		if !filepath.IsAbs(identity) {
			identity = r.resolvePath(identity)
		}
		args = append(args, "-i", identity)
	}
	host := remote.Host
	if remote.User != "" {
		host = remote.User + "@" + remote.Host
	}
	args = append(args, host)
	remoteCmd := buildRemoteCommand(command, env, workdir, shell)
	args = append(args, remoteCmd)
	return exec.CommandContext(ctx, "ssh", args...)
}

func buildRemoteCommand(command string, env []string, workdir string, shell string) string {
	envMap := envListToMap(env)
	kind := shellKind(shell)
	assignments := make([]string, 0, len(envMap))
	for key, value := range envMap {
		if kind == "powershell" {
			assignments = append(assignments, fmt.Sprintf("$env:%s=%s", key, powershellQuote(value)))
		} else {
			assignments = append(assignments, fmt.Sprintf("%s=%s", key, shellQuote(value)))
		}
	}
	sort.Strings(assignments)

	cmd := command
	if workdir != "" {
		if kind == "powershell" {
			cmd = fmt.Sprintf("Set-Location -Path %s; %s", powershellQuote(workdir), cmd)
		} else {
			cmd = fmt.Sprintf("cd %s && %s", shellQuote(workdir), cmd)
		}
	}
	if len(assignments) > 0 {
		joiner := " "
		if kind == "powershell" {
			joiner = "; "
		}
		cmd = strings.Join(assignments, joiner) + joiner + cmd
	}
	return wrapRemoteShell(shell, cmd, kind)
}

func wrapRemoteShell(shell, command, kind string) string {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		return command
	}
	lower := strings.ToLower(shell)
	if strings.Contains(lower, "-c") || strings.Contains(lower, "-command") {
		if kind == "powershell" {
			return shell + " " + powershellQuote(command)
		}
		return shell + " " + shellQuote(command)
	}
	if kind == "powershell" {
		return shell + " -NoProfile -Command " + powershellQuote(command)
	}
	return shell + " -c " + shellQuote(command)
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func powershellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func shellKind(shell string) string {
	lower := strings.ToLower(shell)
	if strings.Contains(lower, "powershell") || strings.Contains(lower, "pwsh") {
		return "powershell"
	}
	return "sh"
}
