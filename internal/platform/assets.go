package platform

import (
	"fmt"
	"runtime"
	"strings"
)

func AssetName(goos, arch, suffix string) (string, error) {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return "", fmt.Errorf("asset suffix is required")
	}
	switch goos {
	case "linux":
		switch arch {
		case "amd64", "arm64":
			return fmt.Sprintf("linux-%s-%s", arch, suffix), nil
		}
	case "darwin":
		switch arch {
		case "amd64", "arm64":
			return fmt.Sprintf("darwin-%s-%s", arch, suffix), nil
		}
	case "windows":
		if arch == "amd64" {
			return fmt.Sprintf("windows-amd64-%s.exe", suffix), nil
		}
	}
	return "", fmt.Errorf("unsupported platform: %s/%s", goos, arch)
}

func BinaryName() string {
	if runtime.GOOS == "windows" {
		return "vbuild.exe"
	}
	return "vbuild"
}
