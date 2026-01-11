package runner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) validateData(task *config.Task, vars map[string]string) error {
	if task == nil || task.Validate == nil {
		return nil
	}
	spec := task.Validate
	if len(spec.Paths) == 0 {
		return fmt.Errorf("validate: paths empty")
	}
	files, err := r.resolvePatterns(spec.Paths, vars)
	if err != nil {
		return err
	}
	count := len(files)
	if spec.MinFiles > 0 && count < spec.MinFiles {
		return fmt.Errorf("validate: expected at least %d files, got %d", spec.MinFiles, count)
	}
	if spec.MaxFiles > 0 && count > spec.MaxFiles {
		return fmt.Errorf("validate: expected at most %d files, got %d", spec.MaxFiles, count)
	}
	minSize := int64(0)
	maxSize := int64(0)
	if spec.MinSize != "" {
		if parsed, err := parseByteSize(spec.MinSize); err == nil {
			minSize = parsed
		} else {
			return err
		}
	}
	if spec.MaxSize != "" {
		if parsed, err := parseByteSize(spec.MaxSize); err == nil {
			maxSize = parsed
		} else {
			return err
		}
	}
	exts := map[string]struct{}{}
	for _, ext := range spec.Extensions {
		exts[strings.ToLower(strings.TrimSpace(ext))] = struct{}{}
	}
	var sampleRe *regexp.Regexp
	if spec.SampleRegex != "" {
		re, err := regexp.Compile(spec.SampleRegex)
		if err != nil {
			return err
		}
		sampleRe = re
	}
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}
		if minSize > 0 && info.Size() < minSize {
			return fmt.Errorf("validate: %s below min size", file)
		}
		if maxSize > 0 && info.Size() > maxSize {
			return fmt.Errorf("validate: %s above max size", file)
		}
		if len(exts) > 0 {
			ext := strings.ToLower(filepath.Ext(file))
			if _, ok := exts[ext]; !ok {
				return fmt.Errorf("validate: %s has invalid extension", file)
			}
		}
		if sampleRe != nil {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			reader := bufio.NewReader(f)
			line, _ := reader.ReadString('\n')
			_ = f.Close()
			if !sampleRe.MatchString(line) {
				return fmt.Errorf("validate: %s sample does not match regex", file)
			}
		}
	}
	return nil
}
