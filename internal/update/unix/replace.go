package unix

import (
	"fmt"
	"os"
)

func ReplaceBinary(exePath, tempPath string) error {
	backupPath := exePath + ".bak"
	_ = os.Remove(backupPath)
	if err := os.Rename(exePath, backupPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := os.Rename(tempPath, exePath); err != nil {
		_ = os.Rename(backupPath, exePath)
		return fmt.Errorf("replace binary: %w", err)
	}
	_ = os.Remove(backupPath)
	return nil
}
