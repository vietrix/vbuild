package runner

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

func applyLimits(command string, limits *config.ResourceLimits, shell string, allowOnWindows bool) string {
	if limits == nil {
		return command
	}
	if runtime.GOOS == "windows" && !allowOnWindows {
		return command
	}
	if !supportsULimit(shell) {
		return command
	}

	parts := []string{}
	if limits.CPU != "" {
		if duration, err := time.ParseDuration(limits.CPU); err == nil {
			seconds := int64(math.Ceil(duration.Seconds()))
			if seconds > 0 {
				parts = append(parts, fmt.Sprintf("ulimit -t %d", seconds))
			}
		}
	}
	if limits.Memory != "" {
		if bytes, err := parseByteSize(limits.Memory); err == nil {
			kb := bytes / 1024
			if kb > 0 {
				parts = append(parts, fmt.Sprintf("ulimit -v %d", kb))
			}
		}
	}
	if len(parts) == 0 {
		return command
	}
	return strings.Join(parts, " && ") + " && " + command
}

func supportsULimit(shell string) bool {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		return true
	}
	base := strings.ToLower(strings.Fields(shell)[0])
	return strings.Contains(base, "sh") || strings.Contains(base, "bash") || strings.Contains(base, "zsh") || strings.Contains(base, "ksh") || strings.Contains(base, "dash")
}

func parseByteSize(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("empty size")
	}
	trimmed = strings.ToLower(trimmed)

	i := 0
	for i < len(trimmed) && (trimmed[i] == '.' || (trimmed[i] >= '0' && trimmed[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("invalid size: %s", value)
	}
	number := trimmed[:i]
	suffix := strings.TrimSpace(trimmed[i:])
	if suffix == "" {
		suffix = "b"
	}

	factor := int64(1)
	switch suffix {
	case "b":
		factor = 1
	case "k", "kb", "kib":
		factor = 1024
	case "m", "mb", "mib":
		factor = 1024 * 1024
	case "g", "gb", "gib":
		factor = 1024 * 1024 * 1024
	case "t", "tb", "tib":
		factor = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("invalid size suffix: %s", value)
	}

	parsed, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, err
	}
	return int64(parsed * float64(factor)), nil
}
