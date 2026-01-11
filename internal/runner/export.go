package runner

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) exportArtifacts(taskName string, task *config.Task, vars map[string]string) (string, []string, error) {
	if task == nil || task.Export == nil {
		return "", nil, nil
	}
	files, err := r.resolveExportFiles(task, vars)
	if err != nil {
		return "", nil, err
	}
	if len(files) == 0 {
		return "", nil, fmt.Errorf("export: no files to export")
	}
	path := r.resolvePath(expandVars(task.Export.Path, vars))
	format := strings.ToLower(strings.TrimSpace(task.Export.Format))
	if format == "" {
		format = "dir"
	}
	switch format {
	case "dir":
		if err := exportToDir(path, files, r.configRoot); err != nil {
			return "", nil, err
		}
		return path, files, nil
	case "zip":
		if err := exportToZip(path, files, r.configRoot); err != nil {
			return "", nil, err
		}
		return path, files, nil
	case "tar.gz":
		if err := exportToTarGz(path, files, r.configRoot); err != nil {
			return "", nil, err
		}
		return path, files, nil
	default:
		return "", nil, fmt.Errorf("export: unsupported format %s", format)
	}
}

func (r *Runner) resolveExportFiles(task *config.Task, vars map[string]string) ([]string, error) {
	if task == nil || task.Export == nil {
		return nil, nil
	}
	if len(task.Export.Include) > 0 {
		return r.resolvePatterns(task.Export.Include, vars)
	}
	files, err := r.resolveOutputFiles(task, vars)
	if err != nil {
		return nil, err
	}
	for _, out := range task.DatasetOutputs {
		paths, _, err := r.resolveDatasetOutputFiles(out, vars)
		if err != nil {
			return nil, err
		}
		files = append(files, paths...)
	}
	return dedupeNonEmpty(files), nil
}

func exportToDir(dest string, files []string, root string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for _, file := range files {
		rel := relPath(file, root)
		target := filepath.Join(dest, rel)
		if err := copyFile(file, target); err != nil {
			return err
		}
	}
	return nil
}

func exportToZip(dest string, files []string, root string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	zipWriter := zip.NewWriter(out)
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		rel := relPath(file, root)
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			continue
		}
		header.Name = filepath.ToSlash(rel)
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(file)
		if err != nil {
			return err
		}
		if _, err := io.Copy(writer, in); err != nil {
			in.Close()
			return err
		}
		in.Close()
	}
	if err := zipWriter.Close(); err != nil {
		return err
	}
	return nil
}

func exportToTarGz(dest string, files []string, root string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		rel := relPath(file, root)
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			continue
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		in, err := os.Open(file)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, in); err != nil {
			in.Close()
			return err
		}
		in.Close()
	}
	return nil
}

func relPath(path, root string) string {
	if root == "" {
		return filepath.Base(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.Base(path)
	}
	return rel
}
