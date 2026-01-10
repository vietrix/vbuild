package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func fetchChecksum(url, assetName string) (string, error) {
	body, err := downloadText(url)
	if err != nil {
		return "", err
	}

	fields := strings.Fields(body)
	if len(fields) == 0 {
		return "", fmt.Errorf("checksum asset is empty")
	}
	checksum := strings.ToLower(fields[0])
	if len(checksum) != 64 {
		return "", fmt.Errorf("checksum has unexpected length")
	}
	if len(fields) > 1 {
		name := filepath.Base(fields[1])
		if name != assetName {
			return "", fmt.Errorf("checksum asset does not match %s", assetName)
		}
	}
	return checksum, nil
}

func verifyChecksum(path, expected string) error {
	actual, err := sha256File(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch: expected %s got %s", expected, actual)
	}
	return nil
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
