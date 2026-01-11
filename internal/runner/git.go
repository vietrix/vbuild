package runner

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
)

func gitMetadata(root string) map[string]string {
	values := map[string]string{}
	if root == "" {
		return values
	}
	root = filepath.Clean(root)
	values["GIT_SHA"] = gitOutput(root, "rev-parse", "HEAD")
	values["GIT_BRANCH"] = gitOutput(root, "rev-parse", "--abbrev-ref", "HEAD")
	values["GIT_TAG"] = gitOutput(root, "describe", "--tags", "--exact-match")
	for key, value := range values {
		values[key] = strings.TrimSpace(value)
	}
	return values
}

func gitOutput(root string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return ""
	}
	return buf.String()
}
