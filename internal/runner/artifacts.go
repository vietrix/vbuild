package runner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) collectArtifacts(taskName string, task *config.Task, vars map[string]string) error {
	if task == nil || len(task.Artifacts) == 0 {
		return nil
	}
	destRoot := r.artifactsRoot(taskName)
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return fmt.Errorf("create artifacts dir: %w", err)
	}

	paths, err := r.resolvePatterns(task.Artifacts, vars)
	if err != nil {
		return err
	}
	collected := []string{}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			dest := filepath.Join(destRoot, filepath.Base(path))
			if err := copyDir(path, dest); err != nil {
				return err
			}
			collected = append(collected, dest)
			continue
		}
		dest := filepath.Join(destRoot, filepath.Base(path))
		if err := copyFile(path, dest); err != nil {
			return err
		}
		collected = append(collected, dest)
	}
	if err := writeArtifactChecksums(destRoot, collected); err != nil {
		return err
	}
	return nil
}

func (r *Runner) artifactsRoot(taskName string) string {
	dir := r.opts.ArtifactsDir
	if dir == "" {
		dir = r.cfg.ArtifactsDir
	}
	if dir == "" {
		dir = ".vbuild/artifacts"
	}
	return filepath.Join(r.configRoot, dir, sanitizePath(taskName))
}

func copyFile(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func sanitizePath(value string) string {
	out := make([]rune, 0, len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "task"
	}
	return string(out)
}

type artifactChecksum struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type artifactManifest struct {
	Files []artifactChecksum `json:"files"`
}

func writeArtifactChecksums(root string, files []string) error {
	entries := []artifactChecksum{}
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		sum, err := sha256File(file)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			rel = filepath.Base(file)
		}
		entries = append(entries, artifactChecksum{Path: rel, Sha256: sum, Size: info.Size()})
	}
	manifest := artifactManifest{Files: entries}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "checksums.json"), data, 0o644)
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
