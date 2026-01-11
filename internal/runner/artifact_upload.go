package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/vietrix/vbuild/internal/config"
)

func (r *Runner) uploadArtifacts(taskName string, files []string) error {
	if r.cfg == nil || r.cfg.Artifacts == nil || len(files) == 0 {
		return nil
	}
	provider := strings.ToLower(strings.TrimSpace(r.cfg.Artifacts.Provider))
	switch provider {
	case "github":
		return r.uploadArtifactsGitHub(taskName, files, r.cfg.Artifacts)
	case "s3":
		return r.uploadArtifactsS3(taskName, files, r.cfg.Artifacts)
	default:
		return nil
	}
}

type githubRelease struct {
	UploadURL string `json:"upload_url"`
}

func (r *Runner) uploadArtifactsGitHub(taskName string, files []string, cfg *config.ArtifactsUpload) error {
	repo := strings.TrimSpace(cfg.Repo)
	if repo == "" {
		repo = strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY"))
	}
	if repo == "" {
		return fmt.Errorf("artifacts upload: repo required")
	}
	tag := strings.TrimSpace(cfg.Tag)
	if tag == "" {
		tag = strings.TrimSpace(os.Getenv("GITHUB_REF_NAME"))
		if tag == "" {
			ref := strings.TrimSpace(os.Getenv("GITHUB_REF"))
			tag = strings.TrimPrefix(ref, "refs/tags/")
		}
	}
	if tag == "" {
		return fmt.Errorf("artifacts upload: tag required")
	}
	token := strings.TrimSpace(cfg.Token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}
	if token == "" {
		return fmt.Errorf("artifacts upload: token required")
	}

	release, err := fetchGitHubRelease(repo, tag, token)
	if err != nil {
		return err
	}
	uploadURL := strings.Split(release.UploadURL, "{")[0]
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		if err := uploadGitHubAsset(uploadURL, token, file); err != nil {
			return err
		}
	}
	return nil
}

func fetchGitHubRelease(repo, tag, token string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("User-Agent", "vbuild")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("artifacts upload: %s", resp.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if release.UploadURL == "" {
		return nil, fmt.Errorf("artifacts upload: missing upload url")
	}
	return &release, nil
}

func uploadGitHubAsset(uploadURL, token, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	name := filepath.Base(filePath)
	endpoint := uploadURL + "?name=" + url.QueryEscape(name)
	req, err := http.NewRequest(http.MethodPost, endpoint, file)
	if err != nil {
		return err
	}
	req.ContentLength = info.Size()
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("artifacts upload: %s (%s)", resp.Status, bytes.TrimSpace(body))
	}
	return nil
}

func (r *Runner) uploadArtifactsS3(taskName string, files []string, cfg *config.ArtifactsUpload) error {
	endpoint, secure, err := resolveCacheEndpoint("s3", cfg.Endpoint, cfg.Region)
	if err != nil {
		return err
	}
	opts := &minio.Options{
		Creds:  resolveS3Credentials("s3", cfg.AccessKey, cfg.SecretKey, cfg.SessionToken),
		Secure: secure,
		Region: cfg.Region,
	}
	if cfg.PathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint, opts)
	if err != nil {
		return err
	}
	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix == "" {
		prefix = "vbuild/artifacts"
	}
	taskPrefix := sanitizePath(taskName)
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		key := path.Join(prefix, taskPrefix, filepath.Base(file))
		_, err = client.FPutObject(context.Background(), cfg.Bucket, key, file, minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		})
		if err != nil {
			return err
		}
	}
	return nil
}
