package runner

import (
	"fmt"
	"path/filepath"
)

func (r *Runner) RegistryPush() error {
	if r.cfg == nil || r.cfg.Registry == nil || r.cfg.Registry.Path == "" {
		return fmt.Errorf("registry.path is required for push")
	}
	if err := r.validateRegistryType("local"); err != nil {
		return err
	}
	src := r.registryRoot()
	dest := r.resolvePath(r.cfg.Registry.Path)
	if samePath(src, dest) {
		return nil
	}
	return copyDir(src, dest)
}

func (r *Runner) RegistryPull() error {
	if r.cfg == nil || r.cfg.Registry == nil || r.cfg.Registry.Path == "" {
		return fmt.Errorf("registry.path is required for pull")
	}
	if err := r.validateRegistryType("local"); err != nil {
		return err
	}
	src := r.resolvePath(r.cfg.Registry.Path)
	dest := r.registryRoot()
	if samePath(src, dest) {
		return nil
	}
	return copyDir(src, dest)
}

func samePath(a, b string) bool {
	ra, errA := filepath.Abs(a)
	rb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return ra == rb
}
