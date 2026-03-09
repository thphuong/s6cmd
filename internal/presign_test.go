package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePresignedURL(t *testing.T) {
	// Start a fake S3 endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set fake credentials so SDK doesn't look for real ones
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")

	t.Run("GET presigned URL contains bucket and key", func(t *testing.T) {
		url, err := GeneratePresignedURL(context.Background(), PresignOptions{
			Bucket:      "test-bucket",
			Key:         "test-key.txt",
			Method:      "GET",
			ExpiresIn:   15 * time.Minute,
			EndpointURL: server.URL,
			Region:      "us-east-1",
		})
		require.NoError(t, err)
		assert.Contains(t, url, "test-bucket")
		assert.Contains(t, url, "test-key.txt")
		assert.Contains(t, url, "X-Amz-Expires=900")
	})

	t.Run("PUT presigned URL contains bucket and key", func(t *testing.T) {
		url, err := GeneratePresignedURL(context.Background(), PresignOptions{
			Bucket:      "test-bucket",
			Key:         "upload/file.bin",
			Method:      "PUT",
			ExpiresIn:   1 * time.Hour,
			EndpointURL: server.URL,
			Region:      "us-east-1",
		})
		require.NoError(t, err)
		assert.Contains(t, url, "test-bucket")
		assert.Contains(t, url, "upload/file.bin")
		assert.Contains(t, url, "X-Amz-Expires=3600")
	})

	t.Run("GET and PUT produce different URLs", func(t *testing.T) {
		getURL, err := GeneratePresignedURL(context.Background(), PresignOptions{
			Bucket:      "mybucket",
			Key:         "mykey",
			Method:      "GET",
			ExpiresIn:   time.Hour,
			EndpointURL: server.URL,
			Region:      "us-east-1",
		})
		require.NoError(t, err)

		putURL, err := GeneratePresignedURL(context.Background(), PresignOptions{
			Bucket:      "mybucket",
			Key:         "mykey",
			Method:      "PUT",
			ExpiresIn:   time.Hour,
			EndpointURL: server.URL,
			Region:      "us-east-1",
		})
		require.NoError(t, err)
		assert.NotEqual(t, getURL, putURL)
	})

	t.Run("unsupported method returns error", func(t *testing.T) {
		_, err := GeneratePresignedURL(context.Background(), PresignOptions{
			Bucket:      "mybucket",
			Key:         "mykey",
			Method:      "DELETE",
			ExpiresIn:   time.Hour,
			EndpointURL: server.URL,
			Region:      "us-east-1",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported method")
	})
}
