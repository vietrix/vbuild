package update

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vietrix/vbuild/internal/platform"
	updateunix "github.com/vietrix/vbuild/internal/update/unix"
	updatewindows "github.com/vietrix/vbuild/internal/update/windows"
)

type Options struct {
	ToVersion string
	Out       io.Writer
}

func Run(opts Options) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	rel, err := fetchRelease(opts.ToVersion)
	if err != nil {
		return err
	}

	suffixes := []string{assetSuffix(rel.TagName)}
	if opts.ToVersion == "" {
		suffixes = []string{"lastest"}
		if fallback := assetSuffix(rel.TagName); fallback != "lastest" {
			suffixes = append(suffixes, fallback)
		}
	}

	findAsset := func(name string) (string, string) {
		assetURL := ""
		checksumURL := ""
		for _, item := range rel.Assets {
			if item.Name == name {
				assetURL = item.URL
				continue
			}
			if item.Name == name+".sha256" {
				checksumURL = item.URL
			}
		}
		return assetURL, checksumURL
	}

	assetURL := ""
	checksumURL := ""
	assetName := ""
	var candidates []string
	for _, suffix := range suffixes {
		name, err := platform.AssetName(runtime.GOOS, runtime.GOARCH, suffix)
		if err != nil {
			return err
		}
		candidates = append(candidates, name)
		url, checksum := findAsset(name)
		if url != "" {
			assetURL = url
			checksumURL = checksum
			assetName = name
			break
		}
	}
	if assetURL == "" {
		return fmt.Errorf("release %s missing asset %s", rel.TagName, strings.Join(candidates, " or "))
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current binary: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	dir := filepath.Dir(exePath)
	tempFile, err := os.CreateTemp(dir, "vbuild-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	cleanup := func() {
		_ = os.Remove(tempPath)
	}
	if runtime.GOOS != "windows" {
		defer cleanup()
	}

	if err := download(assetURL, tempFile); err != nil {
		tempFile.Close()
		cleanup()
		return err
	}
	if err := tempFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	if checksumURL != "" {
		checksum, err := fetchChecksum(checksumURL, assetName)
		if err != nil {
			cleanup()
			return err
		}
		if err := verifyChecksum(tempPath, checksum); err != nil {
			cleanup()
			return err
		}
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempPath, 0o755); err != nil {
			cleanup()
			return fmt.Errorf("mark executable: %w", err)
		}
		if err := updateunix.ReplaceBinary(exePath, tempPath); err != nil {
			cleanup()
			return err
		}
		fmt.Fprintf(opts.Out, "updated to %s\n", rel.TagName)
		return nil
	}

	if err := updatewindows.ScheduleReplace(exePath, tempPath); err != nil {
		cleanup()
		return err
	}
	fmt.Fprintf(opts.Out, "update scheduled to %s (restart vbuild)\n", rel.TagName)
	return nil
}

func assetSuffix(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "lastest"
	}
	if strings.HasSuffix(tag, "-lastest") {
		return "lastest"
	}
	return tag
}
