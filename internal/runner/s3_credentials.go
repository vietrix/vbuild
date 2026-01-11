package runner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func resolveS3Credentials(provider, accessKey, secretKey, sessionToken string) *credentials.Credentials {
	if accessKey != "" || secretKey != "" || sessionToken != "" {
		return credentials.NewStaticV4(accessKey, secretKey, sessionToken)
	}
	if provider == "gcs" {
		if gcpAccess := os.Getenv("GCP_ACCESS_KEY"); gcpAccess != "" {
			return credentials.NewStaticV4(gcpAccess, os.Getenv("GCP_SECRET_KEY"), os.Getenv("GCP_SESSION_TOKEN"))
		}
	}
	profile := strings.TrimSpace(os.Getenv("AWS_PROFILE"))
	if profile == "" && provider == "gcs" {
		profile = strings.TrimSpace(os.Getenv("GCP_PROFILE"))
	}
	if profile != "" {
		filename := strings.TrimSpace(os.Getenv("AWS_SHARED_CREDENTIALS_FILE"))
		if filename == "" {
			if home, err := os.UserHomeDir(); err == nil {
				filename = filepath.Join(home, ".aws", "credentials")
			}
		}
		if filename != "" {
			return credentials.NewFileAWSCredentials(filename, profile)
		}
	}
	return credentials.NewEnvAWS()
}
