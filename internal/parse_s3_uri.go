package internal

import (
	"fmt"
	"strings"
)

// ParseS3URI parses an S3 URI (s3://bucket/key) into bucket and key components.
func ParseS3URI(uri string) (bucket, key string, err error) {
	if uri == "" {
		return "", "", fmt.Errorf("S3 URI is empty")
	}

	if !strings.HasPrefix(uri, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URI %q: must start with s3://", uri)
	}

	// Strip the s3:// prefix
	path := strings.TrimPrefix(uri, "s3://")
	if path == "" {
		return "", "", fmt.Errorf("invalid S3 URI %q: missing bucket name", uri)
	}

	// Split on first /
	idx := strings.Index(path, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid S3 URI %q: missing object key", uri)
	}

	bucket = path[:idx]
	key = path[idx+1:]

	if bucket == "" {
		return "", "", fmt.Errorf("invalid S3 URI %q: missing bucket name", uri)
	}
	if key == "" {
		return "", "", fmt.Errorf("invalid S3 URI %q: missing object key", uri)
	}

	return bucket, key, nil
}
