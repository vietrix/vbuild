package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type traceRecorder struct {
	path   string
	mu     sync.Mutex
	events []traceEvent
}

type traceEvent struct {
	Timestamp  string `json:"ts"`
	Event      string `json:"event"`
	Task       string `json:"task,omitempty"`
	Command    string `json:"command,omitempty"`
	Status     string `json:"status,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Attempt    int    `json:"attempt,omitempty"`
}

func newTraceRecorder(root, path string) *traceRecorder {
	if strings.TrimSpace(path) == "" {
		return &traceRecorder{}
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	return &traceRecorder{path: path}
}

func (t *traceRecorder) record(event traceEvent) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *traceRecorder) TaskStart(task string) {
	t.record(traceEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Event:     "task_start",
		Task:      task,
	})
}

func (t *traceRecorder) TaskEnd(task, status string, duration time.Duration) {
	t.record(traceEvent{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		Event:      "task_end",
		Task:       task,
		Status:     status,
		DurationMs: duration.Milliseconds(),
	})
}

func (t *traceRecorder) CommandStart(task, command string, attempt int) {
	t.record(traceEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Event:     "command_start",
		Task:      task,
		Command:   command,
		Attempt:   attempt,
	})
}

func (t *traceRecorder) CommandEnd(task, command, status string, duration time.Duration, attempt int) {
	t.record(traceEvent{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		Event:      "command_end",
		Task:       task,
		Command:    command,
		Status:     status,
		DurationMs: duration.Milliseconds(),
		Attempt:    attempt,
	})
}

func (t *traceRecorder) Flush() error {
	if t == nil || t.path == "" {
		return nil
	}
	t.mu.Lock()
	events := append([]traceEvent(nil), t.events...)
	t.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.path, data, 0o644)
}

func (r *Runner) flushTrace() {
	if r.trace == nil {
		return
	}
	if err := r.trace.Flush(); err != nil {
		r.log.Errorf("failed to write timeline: %v\n", err)
	}
}
