package runner

import (
	"os"
	"path/filepath"
)

func (r *Runner) Clean() error {
	path := filepath.Join(r.configRoot, ".vbuild")
	return os.RemoveAll(path)
}
