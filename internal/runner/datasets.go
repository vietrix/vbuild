package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

type datasetEntry struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Path     string   `json:"path,omitempty"`
	Files    []string `json:"files,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Hash     string   `json:"hash,omitempty"`
	Manifest string   `json:"manifest,omitempty"`
	Updated  string   `json:"updated,omitempty"`
}

type datasetRegistry struct {
	Datasets map[string][]datasetEntry `json:"datasets"`
}

func (r *Runner) resolveDatasetRefs(task *config.Task, vars map[string]string) (map[string]string, []datasetEntry, error) {
	out := map[string]string{}
	entries := []datasetEntry{}
	if task == nil || len(task.Datasets) == 0 {
		return out, entries, nil
	}
	r.registryMu.Lock()
	defer r.registryMu.Unlock()
	registry, err := r.loadDatasetRegistry()
	if err != nil {
		return nil, nil, err
	}
	for _, ref := range task.Datasets {
		name := strings.TrimSpace(ref.Name)
		if name == "" {
			continue
		}
		entry, ok := r.findDatasetEntry(name, ref.Version, registry)
		if !ok {
			if def, exists := r.cfg.Datasets[name]; exists && def != nil {
				entry, err = r.registerDatasetDefinition(name, def, registry)
				if err != nil {
					return nil, nil, err
				}
				ok = true
			}
		}
		if !ok {
			if ref.Optional {
				continue
			}
			return nil, nil, fmt.Errorf("dataset %s not found", name)
		}
		path := entry.Path
		if path != "" && !filepath.IsAbs(path) {
			path = filepath.Join(r.configRoot, path)
		}
		prefix := "DATASET_" + strings.ToUpper(sanitizeEnvKey(name))
		out[prefix+"_PATH"] = path
		out[prefix+"_VERSION"] = entry.Version
		if entry.Hash != "" {
			out[prefix+"_HASH"] = entry.Hash
		}
		if entry.Manifest != "" {
			out[prefix+"_MANIFEST"] = entry.Manifest
		}
		entries = append(entries, entry)
	}
	return out, entries, nil
}

func (r *Runner) registerDatasetOutputs(taskName string, task *config.Task, vars map[string]string) ([]datasetEntry, error) {
	if task == nil || len(task.DatasetOutputs) == 0 {
		return nil, nil
	}
	r.registryMu.Lock()
	defer r.registryMu.Unlock()
	registry, err := r.loadDatasetRegistry()
	if err != nil {
		return nil, err
	}
	entries := []datasetEntry{}
	for _, out := range task.DatasetOutputs {
		name := strings.TrimSpace(out.Name)
		if name == "" {
			continue
		}
		files, path, err := r.resolveDatasetOutputFiles(out, vars)
		if err != nil {
			return nil, err
		}
		version := strings.TrimSpace(out.Version)
		hash := ""
		if version == "" {
			hash, err = hashFiles(files, r.configRoot)
			if err != nil {
				return nil, err
			}
			version = "sha256-" + hash[:12]
		}
		entry := datasetEntry{
			Name:    name,
			Version: version,
			Path:    path,
			Files:   relativePaths(files, r.configRoot),
			Tags:    append([]string(nil), out.Tags...),
			Hash:    hash,
			Updated: time.Now().UTC().Format(time.RFC3339),
		}
		registry = upsertDatasetEntry(registry, entry)
		if err := r.writeDatasetManifest(entry); err != nil {
			return nil, err
		}
		r.recordDatasetVars(taskName, entry)
		entries = append(entries, entry)
	}
	if err := r.saveDatasetRegistry(registry); err != nil {
		return entries, err
	}
	return entries, nil
}

func (r *Runner) registerDatasetDefinition(name string, dataset *config.Dataset, registry datasetRegistry) (datasetEntry, error) {
	files, path, err := r.resolveDatasetDefinitionFiles(dataset)
	if err != nil {
		return datasetEntry{}, err
	}
	version := strings.TrimSpace(dataset.Version)
	hash := ""
	if version == "" {
		hash, err = hashFiles(files, r.configRoot)
		if err != nil {
			return datasetEntry{}, err
		}
		version = "sha256-" + hash[:12]
	}
	entry := datasetEntry{
		Name:     name,
		Version:  version,
		Path:     path,
		Files:    relativePaths(files, r.configRoot),
		Tags:     append([]string(nil), dataset.Tags...),
		Hash:     hash,
		Manifest: dataset.Manifest,
		Updated:  time.Now().UTC().Format(time.RFC3339),
	}
	registry = upsertDatasetEntry(registry, entry)
	if err := r.writeDatasetManifest(entry); err != nil {
		return datasetEntry{}, err
	}
	if err := r.saveDatasetRegistry(registry); err != nil {
		return datasetEntry{}, err
	}
	return entry, nil
}

func (r *Runner) resolveDatasetDefinitionFiles(dataset *config.Dataset) ([]string, string, error) {
	if dataset == nil {
		return nil, "", nil
	}
	if dataset.Manifest != "" {
		if manifest, err := r.readDatasetManifest(dataset.Manifest); err == nil && manifest != nil {
			if len(manifest.Files) > 0 {
				files := joinRoot(manifest.Files, r.configRoot)
				return files, dataset.Path, nil
			}
			if manifest.Path != "" {
				return r.resolveDatasetPath(manifest.Path)
			}
		}
	}
	if dataset.Path != "" {
		return r.resolveDatasetPath(dataset.Path)
	}
	if len(dataset.Files) > 0 {
		files, err := r.resolvePatterns(dataset.Files, map[string]string{})
		return files, "", err
	}
	return nil, "", nil
}

func (r *Runner) resolveDatasetOutputFiles(out config.DatasetOutput, vars map[string]string) ([]string, string, error) {
	if out.Path != "" {
		pattern := expandVars(out.Path, vars)
		files, path, err := r.resolveDatasetPath(pattern)
		return files, path, err
	}
	if len(out.Files) > 0 {
		files, err := r.resolvePatterns(out.Files, vars)
		return files, "", err
	}
	return nil, "", nil
}

func (r *Runner) resolveDatasetPath(path string) ([]string, string, error) {
	paths, err := r.resolvePatterns([]string{path}, map[string]string{})
	if err != nil {
		return nil, "", err
	}
	return paths, path, nil
}

func (r *Runner) recordDatasetVars(taskName string, entry datasetEntry) {
	prefix := "DATASET_" + strings.ToUpper(sanitizeEnvKey(entry.Name))
	values := map[string]string{
		prefix + "_VERSION": entry.Version,
	}
	if entry.Path != "" {
		values[prefix+"_PATH"] = entry.Path
	}
	if entry.Hash != "" {
		values[prefix+"_HASH"] = entry.Hash
	}
	if entry.Manifest != "" {
		values[prefix+"_MANIFEST"] = entry.Manifest
	}
	r.outputsMu.Lock()
	if r.outputs[taskName] == nil {
		r.outputs[taskName] = map[string]string{}
	}
	for key, value := range values {
		r.outputs[taskName][key] = value
	}
	r.outputsMu.Unlock()
}

func (r *Runner) loadDatasetRegistry() (datasetRegistry, error) {
	registry := datasetRegistry{Datasets: map[string][]datasetEntry{}}
	if err := r.ensureRegistryDir(); err != nil {
		return registry, err
	}
	path := filepath.Join(r.registryRoot(), "datasets.json")
	if err := readJSON(path, &registry); err != nil {
		if os.IsNotExist(err) {
			return registry, nil
		}
		return registry, err
	}
	if registry.Datasets == nil {
		registry.Datasets = map[string][]datasetEntry{}
	}
	return registry, nil
}

func (r *Runner) ListDatasets() ([]datasetEntry, error) {
	r.registryMu.Lock()
	defer r.registryMu.Unlock()
	registry, err := r.loadDatasetRegistry()
	if err != nil {
		return nil, err
	}
	out := []datasetEntry{}
	for _, list := range registry.Datasets {
		out = append(out, list...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Version < out[j].Version
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (r *Runner) GetDataset(name, version string) (datasetEntry, error) {
	r.registryMu.Lock()
	defer r.registryMu.Unlock()
	registry, err := r.loadDatasetRegistry()
	if err != nil {
		return datasetEntry{}, err
	}
	entry, ok := r.findDatasetEntry(name, version, registry)
	if !ok {
		return datasetEntry{}, fmt.Errorf("dataset %s not found", name)
	}
	return entry, nil
}

func (r *Runner) saveDatasetRegistry(registry datasetRegistry) error {
	if err := r.ensureRegistryDir(); err != nil {
		return err
	}
	path := filepath.Join(r.registryRoot(), "datasets.json")
	return writeJSONFile(path, registry)
}

func (r *Runner) findDatasetEntry(name, version string, registry datasetRegistry) (datasetEntry, bool) {
	entries := registry.Datasets[name]
	if len(entries) == 0 {
		return datasetEntry{}, false
	}
	if version == "" {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Updated > entries[j].Updated
		})
		return entries[0], true
	}
	for _, entry := range entries {
		if entry.Version == version {
			return entry, true
		}
	}
	return datasetEntry{}, false
}

func upsertDatasetEntry(registry datasetRegistry, entry datasetEntry) datasetRegistry {
	if registry.Datasets == nil {
		registry.Datasets = map[string][]datasetEntry{}
	}
	list := registry.Datasets[entry.Name]
	replaced := false
	for i, item := range list {
		if item.Version == entry.Version {
			list[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, entry)
	}
	registry.Datasets[entry.Name] = list
	return registry
}

type datasetManifest struct {
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Path    string   `json:"path,omitempty"`
	Files   []string `json:"files,omitempty"`
	Hash    string   `json:"hash,omitempty"`
}

func (r *Runner) writeDatasetManifest(entry datasetEntry) error {
	manifestPath := filepath.Join(r.registryRoot(), "datasets", sanitizePath(entry.Name), entry.Version+".json")
	manifest := datasetManifest{
		Name:    entry.Name,
		Version: entry.Version,
		Path:    entry.Path,
		Files:   entry.Files,
		Hash:    entry.Hash,
	}
	return writeJSONFile(manifestPath, manifest)
}

func (r *Runner) readDatasetManifest(path string) (*datasetManifest, error) {
	if path == "" {
		return nil, fmt.Errorf("manifest path is empty")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(r.configRoot, path)
	}
	var manifest datasetManifest
	if err := readJSON(path, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func hashFiles(files []string, root string) (string, error) {
	hash := sha256.New()
	for _, file := range files {
		path := file
		if root != "" && !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write([]byte(filepath.ToSlash(file)))
		_, _ = hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func relativePaths(paths []string, root string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}

func joinRoot(paths []string, root string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if filepath.IsAbs(path) {
			out = append(out, path)
		} else {
			out = append(out, filepath.Join(root, path))
		}
	}
	return out
}

func sanitizeEnvKey(value string) string {
	value = strings.ReplaceAll(value, ":", "_")
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, ".", "_")
	return value
}
