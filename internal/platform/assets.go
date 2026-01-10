package platform

import (
	"fmt"
	"runtime"
)

func AssetName(goos, arch string) (string, error) {
	switch goos {
	case "linux":
		switch arch {
		case "amd64", "arm64":
			return fmt.Sprintf("linux-%s", arch), nil
		}
	case "darwin":
		switch arch {
		case "amd64", "arm64":
			return fmt.Sprintf("darwin-%s", arch), nil
		}
	case "windows":
		if arch == "amd64" {
			return "windows-amd64.exe", nil
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
