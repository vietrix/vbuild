package runner

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) runSplit(taskName string, task *config.Task, vars map[string]string, seed int64) error {
	if task == nil || task.Split == nil {
		return nil
	}
	spec := task.Split
	input := expandVars(spec.Input, vars)
	output := expandVars(spec.Output, vars)
	if input == "" || output == "" {
		return fmt.Errorf("split: input/output required")
	}
	files, err := r.resolvePatterns([]string{input}, vars)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("split: no files found for %s", input)
	}
	sortPaths(files)
	total := float64(len(files))
	trainCount := int(math.Floor(total * spec.Train))
	valCount := int(math.Floor(total * spec.Val))
	testCount := int(math.Floor(total * spec.Test))
	remaining := len(files) - trainCount - valCount - testCount
	if remaining > 0 {
		trainCount += remaining
	}
	if spec.Shuffle {
		rngSeed := seed
		if spec.Seed != 0 {
			rngSeed = spec.Seed
		}
		if rngSeed == 0 {
			rngSeed = time.Now().UnixNano()
		}
		rng := rand.New(rand.NewSource(rngSeed))
		rng.Shuffle(len(files), func(i, j int) {
			files[i], files[j] = files[j], files[i]
		})
	}
	train := files[:min(trainCount, len(files))]
	val := files[min(trainCount, len(files)):min(trainCount+valCount, len(files))]
	test := files[min(trainCount+valCount, len(files)):]

	format := strings.ToLower(strings.TrimSpace(spec.Format))
	if format == "" {
		format = "list"
	}
	outDir := r.resolvePath(output)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	switch format {
	case "copy":
		if err := copySplitFiles(outDir, "train", train); err != nil {
			return err
		}
		if err := copySplitFiles(outDir, "val", val); err != nil {
			return err
		}
		if err := copySplitFiles(outDir, "test", test); err != nil {
			return err
		}
	default:
		if err := writeSplitList(outDir, "train.txt", train, r.configRoot); err != nil {
			return err
		}
		if err := writeSplitList(outDir, "val.txt", val, r.configRoot); err != nil {
			return err
		}
		if err := writeSplitList(outDir, "test.txt", test, r.configRoot); err != nil {
			return err
		}
	}
	return nil
}

func writeSplitList(root, name string, files []string, base string) error {
	path := filepath.Join(root, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, item := range files {
		rel := item
		if base != "" {
			if relPath, err := filepath.Rel(base, item); err == nil {
				rel = relPath
			}
		}
		if _, err := writer.WriteString(filepath.ToSlash(rel) + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func copySplitFiles(root, label string, files []string) error {
	dest := filepath.Join(root, label)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for _, file := range files {
		target := filepath.Join(dest, filepath.Base(file))
		if err := copyFile(file, target); err != nil {
			return err
		}
	}
	return nil
}

func sortPaths(paths []string) {
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[j] < paths[i] {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
