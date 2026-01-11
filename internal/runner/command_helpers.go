package runner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
)

type outputCapture struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
}

func newOutputCapture(limit int) *outputCapture {
	if limit <= 0 {
		limit = 256 * 1024
	}
	return &outputCapture{limit: limit}
}

func (c *outputCapture) Write(p []byte) (int, error) {
	length := len(p)
	c.mu.Lock()
	defer c.mu.Unlock()
	remaining := c.limit - c.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = c.buf.Write(p)
	}
	return length, nil
}

func (c *outputCapture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

type captureWriter struct {
	out     io.Writer
	capture *outputCapture
}

func newCaptureWriter(out io.Writer, capture *outputCapture) io.Writer {
	return &captureWriter{out: out, capture: capture}
}

func (w *captureWriter) Write(p []byte) (int, error) {
	if w.capture != nil {
		_, _ = w.capture.Write(p)
	}
	return w.out.Write(p)
}

func (w *captureWriter) Flush() {
	if flusher, ok := w.out.(interface{ Flush() }); ok {
		flusher.Flush()
	}
}

func compileRetryRegex(patterns []string) ([]*regexp.Regexp, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("retry_on_regex invalid: %w", err)
		}
		out = append(out, re)
	}
	return out, nil
}

func shouldRetryCommand(err error, output string, codes []int, regexes []*regexp.Regexp) bool {
	if len(codes) == 0 && len(regexes) == 0 {
		return true
	}
	exitCode := -1
	var exitErr interface{ ExitCode() int }
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	for _, code := range codes {
		if code == exitCode {
			return true
		}
	}
	for _, re := range regexes {
		if re.MatchString(output) {
			return true
		}
	}
	return false
}

func parseCommandTimeout(command string) (time.Duration, string) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return 0, command
	}
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "timeout=") {
		return 0, command
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return 0, command
	}
	value := strings.TrimSpace(strings.TrimPrefix(parts[0], "timeout="))
	if value == "" {
		return 0, command
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, command
	}
	cleaned := strings.TrimSpace(parts[1])
	return duration, cleaned
}
