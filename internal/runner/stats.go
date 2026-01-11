package runner

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

type statsPayload struct {
	Generated string            `json:"generated"`
	Files     int               `json:"files"`
	Bytes     int64             `json:"bytes"`
	Lines     int64             `json:"lines,omitempty"`
	Hashes    map[string]string `json:"hashes,omitempty"`
}

func (r *Runner) writeStats(taskName string, task *config.Task, vars map[string]string) error {
	if task == nil || task.Stats == nil {
		return nil
	}
	spec := task.Stats
	files, err := r.resolvePatterns(spec.Paths, vars)
	if err != nil {
		return err
	}
	payload := statsPayload{
		Generated: time.Now().UTC().Format(time.RFC3339),
		Hashes:    map[string]string{},
	}
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		payload.Files++
		payload.Bytes += info.Size()
		if spec.Lines {
			lines, err := countLines(file)
			if err == nil {
				payload.Lines += lines
			}
		}
		if spec.Hash {
			sum, err := sha256FileLocal(file)
			if err == nil {
				rel := file
				if r.configRoot != "" {
					if relPath, err := filepath.Rel(r.configRoot, file); err == nil {
						rel = relPath
					}
				}
				payload.Hashes[filepath.ToSlash(rel)] = sum
			}
		}
	}
	if !spec.Hash {
		payload.Hashes = nil
	}
	out := spec.Output
	if out == "" {
		out = filepath.Join(".vbuild", "stats", sanitizePath(taskName)+".json")
	}
	path := r.resolvePath(expandVars(out, vars))
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func countLines(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var count int64
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func sha256FileLocal(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	buf := make([]byte, 32*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			_, _ = hash.Write(buf[:n])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
