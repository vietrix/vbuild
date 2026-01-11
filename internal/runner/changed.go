package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

type ChangedOptions struct {
	BaseRef          string
	TargetRef        string
	IncludeUntracked bool
}

func (r *Runner) ChangedTargets(targets []string, opts ChangedOptions) ([]string, error) {
	plan, err := r.buildPlan(r.allTaskNames())
	if err != nil {
		return nil, err
	}
	changed, err := gitChangedFiles(r.configRoot, opts)
	if err != nil {
		return nil, err
	}
	if len(changed) == 0 {
		return []string{}, nil
	}

	changedSet := map[string]struct{}{}
	for _, file := range changed {
		changedSet[file] = struct{}{}
	}

	selected := []string{}
	targetSet := map[string]struct{}{}
	for _, t := range targets {
		targetSet[t] = struct{}{}
	}

	for name, task := range plan.tasks {
		if len(targetSet) > 0 {
			if _, ok := targetSet[name]; !ok {
				continue
			}
		}
		vars := r.taskVars(plan, name, task)
		patterns := append([]string{}, task.Inputs...)
		patterns = append(patterns, task.Watch...)
		if len(patterns) == 0 {
			continue
		}
		if matchesChanged(patterns, vars, r.configRoot, changedSet) {
			selected = append(selected, name)
		}
	}
	out := dedupeNonEmpty(selected)
	sort.Strings(out)
	return out, nil
}

func matchesChanged(patterns []string, vars map[string]string, root string, changed map[string]struct{}) bool {
	includes := []string{}
	excludes := []string{}
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.HasPrefix(pattern, "!") {
			excludes = append(excludes, strings.TrimSpace(strings.TrimPrefix(pattern, "!")))
			continue
		}
		includes = append(includes, pattern)
	}

	for file := range changed {
		abs := filepath.Join(root, filepath.FromSlash(file))
		matched := false
		for _, pattern := range includes {
			pattern = expandVars(pattern, vars)
			pattern = normalizePattern(root, pattern)
			if ok, _ := doublestar.Match(pattern, filepath.ToSlash(abs)); ok {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		excluded := false
		for _, pattern := range excludes {
			pattern = expandVars(pattern, vars)
			pattern = normalizePattern(root, pattern)
			if ok, _ := doublestar.Match(pattern, filepath.ToSlash(abs)); ok {
				excluded = true
				break
			}
		}
		if !excluded {
			return true
		}
	}
	return false
}

func normalizePattern(root, pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return pattern
	}
	if filepath.IsAbs(pattern) || isURL(pattern) {
		return filepath.ToSlash(pattern)
	}
	return filepath.ToSlash(filepath.Join(root, pattern))
}

func gitChangedFiles(root string, opts ChangedOptions) ([]string, error) {
	base := strings.TrimSpace(opts.BaseRef)
	target := strings.TrimSpace(opts.TargetRef)
	args := []string{"diff", "--name-only"}
	if base != "" && target != "" {
		args = append(args, base, target)
	} else if base != "" {
		args = append(args, base)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}
	files := parseGitList(output)

	if opts.IncludeUntracked {
		untracked, err := gitUntrackedFiles(root)
		if err != nil {
			return nil, err
		}
		files = append(files, untracked...)
	}

	out := []string{}
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		out = append(out, filepath.ToSlash(file))
	}
	return dedupeNonEmpty(out), nil
}

func gitUntrackedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}
	return parseGitList(output), nil
}

func parseGitList(output []byte) []string {
	lines := bytes.Split(output, []byte{'\n'})
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(string(line))
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func (r *Runner) ChangedSince(targets []string, since time.Time) ([]string, error) {
	plan, err := r.buildPlan(r.allTaskNames())
	if err != nil {
		return nil, err
	}
	selected := []string{}
	targetSet := map[string]struct{}{}
	for _, t := range targets {
		targetSet[t] = struct{}{}
	}
	for name, task := range plan.tasks {
		if len(targetSet) > 0 {
			if _, ok := targetSet[name]; !ok {
				continue
			}
		}
		vars := r.taskVars(plan, name, task)
		patterns := append([]string{}, task.Inputs...)
		patterns = append(patterns, task.Watch...)
		if len(patterns) == 0 {
			continue
		}
		changed, err := r.patternsSince(patterns, vars, since)
		if err != nil {
			return nil, err
		}
		if changed {
			selected = append(selected, name)
		}
	}
	out := dedupeNonEmpty(selected)
	sort.Strings(out)
	return out, nil
}

func (r *Runner) patternsSince(patterns []string, vars map[string]string, since time.Time) (bool, error) {
	files, err := r.resolvePatterns(patterns, vars)
	if err != nil {
		return false, err
	}
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().After(since) {
			return true, nil
		}
	}
	return false, nil
}
