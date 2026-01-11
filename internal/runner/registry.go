package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func (r *Runner) registryRoot() string {
	if r.cfg != nil && r.cfg.Registry != nil && r.cfg.Registry.Path != "" {
		if filepath.IsAbs(r.cfg.Registry.Path) {
			return r.cfg.Registry.Path
		}
		return filepath.Join(r.configRoot, r.cfg.Registry.Path)
	}
	return filepath.Join(r.configRoot, ".vbuild", "registry")
}

func (r *Runner) RegistryRoot() string {
	return r.registryRoot()
}

func (r *Runner) ensureRegistryDir() error {
	path := r.registryRoot()
	return os.MkdirAll(path, 0o755)
}

func readJSON(path string, value interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func writeJSONFile(path string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (r *Runner) validateRegistryType(expected string) error {
	if r.cfg == nil || r.cfg.Registry == nil || r.cfg.Registry.Type == "" {
		return nil
	}
	if r.cfg.Registry.Type != expected {
		return fmt.Errorf("registry type %q not supported (expected %s)", r.cfg.Registry.Type, expected)
	}
	return nil
}
