package runner

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type commandError struct {
	cmd  string
	code int
	err  error
}

func (e *commandError) Error() string {
	return fmt.Sprintf("command failed (%d): %s", e.code, e.cmd)
}

func (e *commandError) Unwrap() error {
	return e.err
}

func (e *commandError) ExitCode() int {
	return e.code
}

func mergeEnv(base []string, overlays ...map[string]string) []string {
	merged := map[string]string{}
	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			merged[parts[0]] = parts[1]
		}
	}

	for _, overlay := range overlays {
		for key, value := range overlay {
			merged[key] = value
		}
	}

	out := make([]string, 0, len(merged))
	for key, value := range merged {
		out = append(out, key+"="+value)
	}
	return out
}

var varPattern = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_]+)\s*\}\}`)

func expandVars(command string, vars map[string]string) string {
	if len(vars) == 0 {
		return command
	}
	return varPattern.ReplaceAllStringFunc(command, func(match string) string {
		parts := varPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		if value, ok := vars[parts[1]]; ok {
			return value
		}
		return match
	})
}

func formatDuration(duration time.Duration) string {
	return duration.Round(time.Millisecond).String()
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr interface{ ExitCode() int }
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}
