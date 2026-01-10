package windows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func ScheduleReplace(exePath, newPath string) error {
	scriptPath := filepath.Join(filepath.Dir(exePath), "vbuild-update.cmd")
	backupPath := exePath + ".bak"
	script := fmt.Sprintf("@echo off\r\nset PID=%d\r\n:wait\r\ntasklist /FI \"PID eq %d\" | find \"%d\" >NUL\r\nif \"%%ERRORLEVEL%%\"==\"0\" (\r\n  timeout /t 1 /nobreak >NUL\r\n  goto wait\r\n)\r\nif exist \"%s\" del /f /q \"%s\" >NUL\r\nmove /y \"%s\" \"%s\" >NUL\r\nif not \"%%ERRORLEVEL%%\"==\"0\" goto cleanup\r\nmove /y \"%s\" \"%s\" >NUL\r\nif not \"%%ERRORLEVEL%%\"==\"0\" goto rollback\r\ndel \"%s\" >NUL\r\ngoto cleanup\r\n:rollback\r\nmove /y \"%s\" \"%s\" >NUL\r\n:cleanup\r\ndel \"%s\" >NUL\r\n", os.Getpid(), os.Getpid(), os.Getpid(), backupPath, backupPath, exePath, backupPath, newPath, exePath, backupPath, backupPath, exePath, scriptPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return fmt.Errorf("write update script: %w", err)
	}

	cmd := exec.Command("cmd", "/C", scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update script: %w", err)
	}
	return nil
}
