package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

type sbomEntry struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type sbomPayload struct {
	Files []sbomEntry `json:"files"`
}

func (r *Runner) writeSBOM(task *config.Task, vars map[string]string, files []string) (string, error) {
	if task == nil || task.SBOM == nil {
		return "", nil
	}
	if len(files) == 0 {
		outputs, err := r.resolveOutputFiles(task, vars)
		if err != nil {
			return "", err
		}
		files = outputs
	}
	if len(files) == 0 {
		return "", fmt.Errorf("sbom: no files to include")
	}
	path := task.SBOM.Path
	if path == "" {
		path = filepath.Join(".vbuild", "sbom", "sbom.json")
	}
	path = r.resolvePath(expandVars(path, vars))
	format := strings.ToLower(strings.TrimSpace(task.SBOM.Format))
	if format == "" {
		format = "json"
	}
	entries := make([]sbomEntry, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		sum, err := sha256File(file)
		if err != nil {
			return "", err
		}
		rel := relPath(file, r.configRoot)
		entries = append(entries, sbomEntry{Path: filepath.ToSlash(rel), Sha256: sum, Size: info.Size()})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	switch format {
	case "json":
		payload := sbomPayload{Files: entries}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return "", err
		}
	case "txt":
		var builder strings.Builder
		for _, entry := range entries {
			builder.WriteString(fmt.Sprintf("%s  %s\n", entry.Sha256, entry.Path))
		}
		if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("sbom: unsupported format %s", format)
	}
	return path, nil
}
