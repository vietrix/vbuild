package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fsnotify/fsnotify"
	"github.com/vietrix/vbuild/internal/config"
)

type WatchOptions struct {
	Interval  time.Duration
	Debounce  time.Duration
	UseEvents bool
}

type fileSnapshot struct {
	ModTime int64
	Size    int64
}

func (r *Runner) Watch(taskName string, opts WatchOptions) error {
	if taskName == "" {
		return fmt.Errorf("task name is required")
	}
	task, ok := r.cfg.Tasks[taskName]
	if !ok || task == nil {
		return fmt.Errorf("unknown task: %s", taskName)
	}
	vars := r.taskVars(&taskPlan{variants: map[string]taskVariant{}}, taskName, task)
	patterns := watchPatterns(task)
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	r.log.Printf("==> watching %s\n", taskName)
	if opts.UseEvents {
		if err := r.watchEvents(taskName, patterns, vars, opts.Debounce); err == nil {
			return nil
		} else {
			r.log.Errorf("watch events failed: %v; falling back to polling\n", err)
		}
	}
	return r.watchPoll(taskName, patterns, vars, opts.Interval, opts.Debounce)
}

func watchPatterns(task *config.Task) []string {
	if len(task.Watch) > 0 {
		return task.Watch
	}
	if len(task.Inputs) > 0 {
		return task.Inputs
	}
	return nil
}

func (r *Runner) watchPoll(taskName string, patterns []string, vars map[string]string, interval, debounce time.Duration) error {
	snapshot, err := r.buildSnapshot(patterns, vars)
	if err != nil {
		return err
	}
	for {
		time.Sleep(interval)
		next, err := r.buildSnapshot(patterns, vars)
		if err != nil {
			r.log.Errorf("watch error: %v\n", err)
			continue
		}
		if !snapshotChanged(snapshot, next) {
			continue
		}
		if debounce > 0 {
			for {
				time.Sleep(debounce)
				latest, err := r.buildSnapshot(patterns, vars)
				if err != nil {
					r.log.Errorf("watch error: %v\n", err)
					break
				}
				if !snapshotChanged(next, latest) {
					next = latest
					break
				}
				next = latest
			}
		}
		snapshot = next
		_ = r.RunTargets([]string{taskName})
	}
}

func (r *Runner) watchEvents(taskName string, patterns []string, vars map[string]string, debounce time.Duration) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	roots, err := r.watchRoots(patterns, vars)
	if err != nil {
		return err
	}
	for _, root := range roots {
		if err := addWatchRecursive(watcher, root); err != nil {
			return err
		}
	}

	trigger := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = addWatchRecursive(watcher, event.Name)
					}
				}
				if matchWatchPatterns(patterns, vars, r.configRoot, event.Name) {
					select {
					case trigger <- struct{}{}:
					default:
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				r.log.Errorf("watch error: %v\n", err)
			}
		}
	}()

	var timer *time.Timer
	for {
		<-trigger
		if debounce == 0 {
			_ = r.RunTargets([]string{taskName})
			continue
		}
		if timer != nil {
			timer.Stop()
		}
		timer = time.NewTimer(debounce)
		for {
			select {
			case <-timer.C:
				_ = r.RunTargets([]string{taskName})
				timer = nil
				goto next
			case <-trigger:
				if timer != nil {
					timer.Stop()
				}
				timer = time.NewTimer(debounce)
			}
		}
	next:
	}
}

func (r *Runner) watchRoots(patterns []string, vars map[string]string) ([]string, error) {
	roots := []string{}
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || strings.HasPrefix(pattern, "!") {
			continue
		}
		expanded := expandVars(pattern, vars)
		expanded = r.resolvePath(expanded)
		root := globRoot(expanded)
		if root == "" {
			root = r.configRoot
		}
		roots = append(roots, root)
	}
	if len(roots) == 0 {
		roots = append(roots, r.configRoot)
	}
	return dedupeNonEmpty(roots), nil
}

func globRoot(pattern string) string {
	idx := strings.IndexAny(pattern, "*?[")
	if idx == -1 {
		return pattern
	}
	prefix := filepath.Clean(pattern[:idx])
	if prefix == "." || prefix == string(filepath.Separator) {
		return prefix
	}
	return filepath.Dir(prefix)
}

func addWatchRecursive(watcher *fsnotify.Watcher, root string) error {
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

func matchWatchPatterns(patterns []string, vars map[string]string, root, path string) bool {
	if len(patterns) == 0 {
		return true
	}
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
	abs := filepath.ToSlash(path)
	matched := len(includes) == 0
	if !matched {
		for _, pattern := range includes {
			pattern = expandVars(pattern, vars)
			pattern = normalizePattern(root, pattern)
			if ok, _ := doublestar.Match(pattern, abs); ok {
				matched = true
				break
			}
		}
	}
	if !matched {
		return false
	}
	for _, pattern := range excludes {
		pattern = expandVars(pattern, vars)
		pattern = normalizePattern(root, pattern)
		if ok, _ := doublestar.Match(pattern, abs); ok {
			return false
		}
	}
	return true
}

func (r *Runner) buildSnapshot(patterns []string, vars map[string]string) (map[string]fileSnapshot, error) {
	files, err := r.resolvePatterns(patterns, vars)
	if err != nil {
		return nil, err
	}
	snapshot := map[string]fileSnapshot{}
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		snapshot[path] = fileSnapshot{ModTime: info.ModTime().UnixNano(), Size: info.Size()}
	}
	return snapshot, nil
}

func snapshotChanged(a, b map[string]fileSnapshot) bool {
	if len(a) != len(b) {
		return true
	}
	for path, info := range a {
		other, ok := b[path]
		if !ok {
			return true
		}
		if info.ModTime != other.ModTime || info.Size != other.Size {
			return true
		}
	}
	return false
}
