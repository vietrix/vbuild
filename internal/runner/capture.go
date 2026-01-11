package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vietrix/vbuild/internal/config"
)

type captureSpec struct {
	Stdout   io.Writer
	Stderr   io.Writer
	Combined io.Writer
	close    []io.Closer
}

func (c *captureSpec) Close() {
	for _, item := range c.close {
		_ = item.Close()
	}
}

func (r *Runner) prepareCapture(task *config.Task, vars map[string]string) (*captureSpec, error) {
	if task == nil || task.Capture == nil {
		return nil, nil
	}
	spec := &captureSpec{}
	appendMode := task.Capture.Append

	openWriter := func(value string) (io.WriteCloser, error) {
		if value == "" {
			return nil, nil
		}
		path := expandVars(value, vars)
		path = r.resolvePath(path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		flags := os.O_CREATE | os.O_WRONLY
		if appendMode {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		return os.OpenFile(path, flags, 0o644)
	}

	if task.Capture.Stdout != "" {
		file, err := openWriter(task.Capture.Stdout)
		if err != nil {
			return nil, fmt.Errorf("capture stdout: %w", err)
		}
		spec.Stdout = file
		spec.close = append(spec.close, file)
	}
	if task.Capture.Stderr != "" {
		file, err := openWriter(task.Capture.Stderr)
		if err != nil {
			return nil, fmt.Errorf("capture stderr: %w", err)
		}
		spec.Stderr = file
		spec.close = append(spec.close, file)
	}
	if task.Capture.Combined != "" {
		file, err := openWriter(task.Capture.Combined)
		if err != nil {
			return nil, fmt.Errorf("capture combined: %w", err)
		}
		spec.Combined = file
		spec.close = append(spec.close, file)
	}
	return spec, nil
}
