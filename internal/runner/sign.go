package runner

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vietrix/vbuild/internal/config"
)

type signatureEntry struct {
	Path      string `json:"path"`
	Sha256    string `json:"sha256"`
	Signature string `json:"signature,omitempty"`
}

type signaturePayload struct {
	Files []signatureEntry `json:"files"`
}

func (r *Runner) writeSignature(task *config.Task, vars map[string]string, files []string, env map[string]string) (string, error) {
	if task == nil || task.Sign == nil {
		return "", nil
	}
	if len(files) == 0 {
		outputs, err := r.resolveOutputFiles(task, vars)
		if err != nil {
			return "", err
		}
		files = outputs
	}
	if len(files) == 0 {
		return "", fmt.Errorf("sign: no files to sign")
	}
	path := task.Sign.Output
	if path == "" {
		path = filepath.Join(".vbuild", "signatures", "signatures.json")
	}
	path = r.resolvePath(expandVars(path, vars))
	key := env["VBUILD_SIGN_KEY"]
	entries := []signatureEntry{}
	for _, file := range files {
		sum, err := sha256File(file)
		if err != nil {
			return "", err
		}
		entry := signatureEntry{Path: filepath.ToSlash(relPath(file, r.configRoot)), Sha256: sum}
		if key != "" {
			mac := hmac.New(sha256.New, []byte(key))
			_, _ = mac.Write([]byte(sum))
			entry.Signature = hex.EncodeToString(mac.Sum(nil))
		}
		entries = append(entries, entry)
	}
	payload := signaturePayload{Files: entries}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
