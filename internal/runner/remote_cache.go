package runner

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/vietrix/vbuild/internal/config"
)

type remoteCache interface {
	Restore(taskName, signature string, inputs map[string]fileFingerprint, root string) (bool, error)
	Store(taskName, signature string, inputs map[string]fileFingerprint, outputs []string, root string) error
}

func (r *Runner) remoteRestore(taskName string, task *config.Task, vars map[string]string, signature string, inputs map[string]fileFingerprint) (bool, error) {
	if r.remote == nil || task == nil {
		return false, nil
	}
	return r.remote.Restore(taskName, signature, inputs, r.configRoot)
}

func (r *Runner) remoteStore(taskName string, task *config.Task, vars map[string]string, signature string, inputs map[string]fileFingerprint) error {
	if r.remote == nil || task == nil {
		return nil
	}
	outputs, err := r.resolveOutputFiles(task, vars)
	if err != nil {
		return err
	}
	if len(outputs) == 0 {
		return nil
	}
	return r.remote.Store(taskName, signature, inputs, outputs, r.configRoot)
}

type remoteCacheEntry struct {
	Signature string                     `json:"signature"`
	Inputs    map[string]fileFingerprint `json:"inputs"`
	Outputs   []string                   `json:"outputs"`
	Created   string                     `json:"created"`
}

type s3RemoteCache struct {
	client     *minio.Client
	bucket     string
	prefix     string
	root       string
	log        *logger
	ensureOnce sync.Once
	ensureErr  error
}

func newRemoteCache(cfg *config.CacheRemote, root string, log *logger) remoteCache {
	if cfg == nil {
		return nil
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		return nil
	}
	endpoint, secure, err := resolveCacheEndpoint(provider, cfg.Endpoint, cfg.Region)
	if err != nil {
		log.Errorf("cache remote: %v\n", err)
		return nil
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		log.Errorf("cache remote: bucket is required\n")
		return nil
	}

	var creds *credentials.Credentials
	if cfg.AccessKey != "" || cfg.SecretKey != "" {
		creds = credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, cfg.SessionToken)
	} else {
		creds = credentials.NewEnvAWS()
	}

	opts := &minio.Options{
		Creds:  creds,
		Secure: secure,
		Region: cfg.Region,
	}
	if cfg.PathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}

	client, err := minio.New(endpoint, opts)
	if err != nil {
		log.Errorf("cache remote: %v\n", err)
		return nil
	}

	prefix := strings.Trim(cfg.Prefix, "/")
	if prefix == "" {
		prefix = "vbuild/cache"
	}
	return &s3RemoteCache{
		client: client,
		bucket: cfg.Bucket,
		prefix: prefix,
		root:   root,
		log:    log,
	}
}

func resolveCacheEndpoint(provider, endpoint, region string) (string, bool, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		switch provider {
		case "s3":
			if region == "" || region == "us-east-1" {
				return "s3.amazonaws.com", true, nil
			}
			return fmt.Sprintf("s3.%s.amazonaws.com", region), true, nil
		case "gcs":
			return "storage.googleapis.com", true, nil
		case "minio":
			return "", false, fmt.Errorf("cache remote: endpoint required for minio")
		default:
			return "", false, fmt.Errorf("cache remote: unsupported provider %q", provider)
		}
	}

	if strings.Contains(endpoint, "://") {
		u, err := url.Parse(endpoint)
		if err != nil {
			return "", false, fmt.Errorf("cache remote: invalid endpoint %q", endpoint)
		}
		if u.Host == "" {
			return "", false, fmt.Errorf("cache remote: invalid endpoint %q", endpoint)
		}
		return u.Host, u.Scheme != "http", nil
	}
	return endpoint, true, nil
}

func (c *s3RemoteCache) ensureBucket(ctx context.Context) error {
	c.ensureOnce.Do(func() {
		exists, err := c.client.BucketExists(ctx, c.bucket)
		if err != nil {
			c.ensureErr = fmt.Errorf("cache remote: bucket check failed: %w", err)
			return
		}
		if !exists {
			c.ensureErr = fmt.Errorf("cache remote: bucket %s not found", c.bucket)
		}
	})
	return c.ensureErr
}

func (c *s3RemoteCache) Restore(taskName, signature string, inputs map[string]fileFingerprint, root string) (bool, error) {
	ctx := context.Background()
	if err := c.ensureBucket(ctx); err != nil {
		return false, err
	}
	metaKey := c.objectKey(taskName, signature, ".json")
	meta, err := c.downloadMeta(ctx, metaKey)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if meta.Signature != signature || !fingerprintsMatch(meta.Inputs, inputs) {
		return false, nil
	}

	archiveKey := c.objectKey(taskName, signature, ".tar.gz")
	tempPath, err := c.downloadObject(ctx, archiveKey)
	if err != nil {
		return false, err
	}
	defer os.Remove(tempPath)
	if err := extractTarGz(tempPath, root); err != nil {
		return false, err
	}
	return true, nil
}

func (c *s3RemoteCache) Store(taskName, signature string, inputs map[string]fileFingerprint, outputs []string, root string) error {
	if len(outputs) == 0 {
		return nil
	}
	ctx := context.Background()
	if err := c.ensureBucket(ctx); err != nil {
		return err
	}

	archivePath, relOutputs, err := createTarGz(outputs, root)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)

	archiveKey := c.objectKey(taskName, signature, ".tar.gz")
	if err := c.uploadFile(ctx, archiveKey, archivePath); err != nil {
		return err
	}

	meta := remoteCacheEntry{
		Signature: signature,
		Inputs:    inputs,
		Outputs:   relOutputs,
		Created:   time.Now().UTC().Format(time.RFC3339Nano),
	}
	metaKey := c.objectKey(taskName, signature, ".json")
	return c.uploadJSON(ctx, metaKey, meta)
}

func (c *s3RemoteCache) objectKey(taskName, signature, suffix string) string {
	name := sanitizePath(taskName)
	return path.Join(c.prefix, name, signature+suffix)
}

func (c *s3RemoteCache) uploadFile(ctx context.Context, key, filePath string) error {
	_, err := c.client.FPutObject(ctx, c.bucket, key, filePath, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("cache remote: upload %s failed: %w", key, err)
	}
	return nil
}

func (c *s3RemoteCache) uploadJSON(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(data)
	_, err = c.client.PutObject(ctx, c.bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("cache remote: upload %s failed: %w", key, err)
	}
	return nil
}

func (c *s3RemoteCache) downloadMeta(ctx context.Context, key string) (*remoteCacheEntry, error) {
	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, err
	}
	var entry remoteCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (c *s3RemoteCache) downloadObject(ctx context.Context, key string) (string, error) {
	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	defer obj.Close()
	if _, err := obj.Stat(); err != nil {
		return "", err
	}
	tempDir := filepath.Join(c.root, ".vbuild", "cache")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", err
	}
	tempFile, err := os.CreateTemp(tempDir, "remote-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	if _, err := io.Copy(tempFile, obj); err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	resp := minio.ToErrorResponse(err)
	if resp.StatusCode == http.StatusNotFound || resp.Code == "NoSuchKey" {
		return true
	}
	return false
}

func createTarGz(outputs []string, root string) (string, []string, error) {
	tempDir := filepath.Join(root, ".vbuild", "cache")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", nil, err
	}
	tempFile, err := os.CreateTemp(tempDir, "cache-*.tar.gz")
	if err != nil {
		return "", nil, err
	}
	defer tempFile.Close()

	gw := gzip.NewWriter(tempFile)
	tw := tar.NewWriter(gw)

	relPaths := []string{}
	for _, output := range outputs {
		info, err := os.Stat(output)
		if err != nil || info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(root, output)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		rel = filepath.ToSlash(rel)
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			continue
		}
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return "", nil, err
		}
		file, err := os.Open(output)
		if err != nil {
			return "", nil, err
		}
		if _, err := io.Copy(tw, file); err != nil {
			file.Close()
			return "", nil, err
		}
		file.Close()
		relPaths = append(relPaths, rel)
	}
	if err := tw.Close(); err != nil {
		return "", nil, err
	}
	if err := gw.Close(); err != nil {
		return "", nil, err
	}
	return tempFile.Name(), relPaths, nil
}

func extractTarGz(path, root string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(filepath.FromSlash(header.Name))
		if name == "." || name == "" {
			continue
		}
		if filepath.IsAbs(name) || strings.HasPrefix(name, "..") {
			return fmt.Errorf("cache remote: invalid path %s", header.Name)
		}
		target := filepath.Join(root, name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			continue
		}
	}
	return nil
}

func (r *Runner) resolveOutputFiles(task *config.Task, vars map[string]string) ([]string, error) {
	patterns := append([]string{}, task.Outputs...)
	patterns = append(patterns, task.OutputPaths...)
	if len(patterns) == 0 {
		return []string{}, nil
	}
	return r.resolvePatterns(dedupeNonEmpty(patterns), vars)
}
