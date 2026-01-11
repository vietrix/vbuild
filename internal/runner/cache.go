package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/vietrix/vbuild/internal/config"
)

type cacheEntry struct {
	Signature string                     `json:"signature"`
	Inputs    map[string]fileFingerprint `json:"inputs"`
	Outputs   map[string]fileFingerprint `json:"outputs"`
}

type fileFingerprint struct {
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
	Hash    string `json:"hash,omitempty"`
}

func (r *Runner) cacheSkip(taskName string, task *config.Task, vars map[string]string) (bool, string) {
	if task == nil {
		return false, ""
	}
	if len(task.Inputs) == 0 && len(task.Outputs) == 0 && len(task.OutputPaths) == 0 {
		return false, ""
	}
	cachePath := r.cachePath(taskName)
	entry, err := readCache(cachePath)
	if err != nil {
		entry = nil
	}
	signature := r.cacheSignature(task, vars)

	inputs, err := r.resolvePatterns(task.Inputs, vars)
	if err != nil {
		return false, ""
	}
	mode := cacheMode(task.Cache)
	inputFingerprints, ok := fingerprintFiles(inputs, mode)
	if !ok {
		return false, ""
	}
	outputsDefined := len(task.Outputs) > 0 || len(task.OutputPaths) > 0
	outputs := []string{}
	outputFingerprints := map[string]fileFingerprint{}
	if outputsDefined {
		outputs, err = r.resolveOutputFiles(task, vars)
		if err != nil {
			return false, ""
		}
		if len(outputs) > 0 {
			outputFingerprints, ok = fingerprintFiles(outputs, mode)
			if !ok {
				outputFingerprints = nil
			}
		}
	}
	if entry != nil && outputFingerprints != nil {
		if entry.Signature == signature &&
			fingerprintsMatch(entry.Inputs, inputFingerprints) &&
			fingerprintsMatch(entry.Outputs, outputFingerprints) {
			return true, "up-to-date"
		}
	}

	if r.remote != nil && outputsDefined {
		restored, err := r.remoteRestore(taskName, task, vars, signature, inputFingerprints)
		if err == nil && restored {
			return true, "remote cache"
		}
	}

	if entry != nil && entry.Signature != signature {
		return false, "signature changed"
	}
	if entry != nil && !fingerprintsMatch(entry.Inputs, inputFingerprints) {
		return false, "inputs changed"
	}
	if entry != nil && outputFingerprints != nil && !fingerprintsMatch(entry.Outputs, outputFingerprints) {
		return false, "outputs changed"
	}
	return false, ""
}

func (r *Runner) cacheSave(taskName string, task *config.Task, vars map[string]string) error {
	if task == nil {
		return nil
	}
	if len(task.Inputs) == 0 && len(task.Outputs) == 0 && len(task.OutputPaths) == 0 {
		return nil
	}
	inputs, err := r.resolvePatterns(task.Inputs, vars)
	if err != nil {
		return err
	}
	outputs, err := r.resolveOutputFiles(task, vars)
	if err != nil {
		return err
	}
	if len(outputs) == 0 {
		return nil
	}

	mode := cacheMode(task.Cache)
	inputFingerprints, ok := fingerprintFiles(inputs, mode)
	if !ok {
		return nil
	}
	outputFingerprints, ok := fingerprintFiles(outputs, mode)
	if !ok {
		return nil
	}

	entry := cacheEntry{
		Signature: r.cacheSignature(task, vars),
		Inputs:    inputFingerprints,
		Outputs:   outputFingerprints,
	}

	cachePath := r.cachePath(taskName)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		return err
	}
	if r.remote != nil {
		_ = r.remoteStore(taskName, task, vars, entry.Signature, inputFingerprints)
	}
	return nil
}

func cacheMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "sha256":
		return "sha256"
	default:
		return "mtime"
	}
}

func (r *Runner) cacheSignature(task *config.Task, vars map[string]string) string {
	commands := append([]string(nil), task.Run...)
	if task.Docker != nil {
		commands = append(commands, dockerCommands(task.Docker, vars)...)
	}
	workdir := task.Workdir
	shell := task.Shell
	timeout := task.Timeout
	backoff := task.Backoff
	jitter := task.Jitter
	retries := task.Retries
	maxRetries := task.MaxRetries
	if r.cfg != nil && r.cfg.Defaults != nil {
		if workdir == "" {
			workdir = r.cfg.Defaults.Workdir
		}
		if shell == "" {
			shell = r.cfg.Defaults.Shell
		}
		if timeout == "" {
			timeout = r.cfg.Defaults.Timeout
		}
		if backoff == "" {
			backoff = r.cfg.Defaults.Backoff
		}
		if jitter == "" {
			jitter = r.cfg.Defaults.Jitter
		}
		if retries == 0 {
			retries = r.cfg.Defaults.Retries
		}
		if maxRetries == 0 {
			maxRetries = r.cfg.Defaults.MaxRetries
		}
	}
	payload := struct {
		Run        []string          `json:"run"`
		Env        map[string]string `json:"env"`
		Vars       map[string]string `json:"vars"`
		Workdir    string            `json:"workdir"`
		Shell      string            `json:"shell"`
		Timeout    string            `json:"timeout"`
		Backoff    string            `json:"backoff"`
		Jitter     string            `json:"jitter"`
		Retries    int               `json:"retries"`
		MaxRetries int               `json:"max_retries"`
		Parallel   bool              `json:"parallel"`
		Signals    []string          `json:"retry_on_signal,omitempty"`
	}{
		Run:        commands,
		Env:        task.Env,
		Vars:       vars,
		Workdir:    workdir,
		Shell:      shell,
		Timeout:    timeout,
		Backoff:    backoff,
		Jitter:     jitter,
		Retries:    retries,
		MaxRetries: maxRetries,
		Parallel:   task.Parallel,
		Signals:    task.RetryOnSignal,
	}
	data, _ := json.Marshal(payload)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (r *Runner) cachePath(taskName string) string {
	return filepath.Join(r.configRoot, ".vbuild", "cache", sanitizePath(taskName)+".json")
}

func readCache(path string) (*cacheEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func fingerprintFiles(paths []string, mode string) (map[string]fileFingerprint, bool) {
	out := map[string]fileFingerprint{}
	for _, path := range paths {
		fp, err := fingerprintFile(path, mode)
		if err != nil {
			return nil, false
		}
		out[path] = fp
	}
	return out, true
}

func fingerprintFile(path string, mode string) (fileFingerprint, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileFingerprint{}, err
	}
	fp := fileFingerprint{Size: info.Size(), ModTime: info.ModTime().UnixNano()}
	if mode == "sha256" && !info.IsDir() {
		hash := sha256.New()
		file, err := os.Open(path)
		if err != nil {
			return fp, err
		}
		defer file.Close()
		if _, err := io.Copy(hash, file); err != nil {
			return fp, err
		}
		fp.Hash = hex.EncodeToString(hash.Sum(nil))
	}
	return fp, nil
}

func fingerprintsMatch(a, b map[string]fileFingerprint) bool {
	if len(a) != len(b) {
		return false
	}
	for key, left := range a {
		right, ok := b[key]
		if !ok {
			return false
		}
		if left.Size != right.Size || left.ModTime != right.ModTime || left.Hash != right.Hash {
			return false
		}
	}
	return true
}

func (r *Runner) resolvePatterns(patterns []string, vars map[string]string) ([]string, error) {
	return r.resolvePatternsWithRoot(patterns, vars, r.configRoot)
}

func (r *Runner) resolvePatternsWithRoot(patterns []string, vars map[string]string, root string) ([]string, error) {
	if len(patterns) == 0 {
		return []string{}, nil
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

	paths := []string{}
	for _, pattern := range includes {
		files, err := r.expandPattern(pattern, vars, root)
		if err != nil {
			return nil, err
		}
		paths = append(paths, files...)
	}

	excluded := map[string]struct{}{}
	for _, pattern := range excludes {
		files, err := r.expandPattern(pattern, vars, root)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			excluded[file] = struct{}{}
		}
	}

	filtered := []string{}
	for _, path := range paths {
		if _, ok := excluded[path]; ok {
			continue
		}
		filtered = append(filtered, path)
	}
	sort.Strings(filtered)
	unique := []string{}
	seen := map[string]struct{}{}
	for _, path := range filtered {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	return unique, nil
}

func (r *Runner) expandPattern(pattern string, vars map[string]string, root string) ([]string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return []string{}, nil
	}
	pattern = expandVars(pattern, vars)
	pattern = r.resolvePathWithRoot(root, pattern)
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		if _, err := os.Stat(pattern); err == nil {
			matches = []string{pattern}
		} else {
			return []string{}, nil
		}
	}

	paths := []string{}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.IsDir() {
			err = filepath.WalkDir(match, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				paths = append(paths, path)
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}
		paths = append(paths, match)
	}
	return paths, nil
}
