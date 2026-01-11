package runner

import (
	"errors"
	"strings"
)

func extractSignal(err error) string {
	var sigErr interface {
		SignalName() string
	}
	if errors.As(err, &sigErr) {
		return normalizeSignal(sigErr.SignalName())
	}
	return ""
}

func normalizeSignal(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "SIG") {
		value = "SIG" + value
	}
	return value
}
