package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runApp creates a fresh app and runs it with the given args, capturing output.
func runApp(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	app := NewApp()
	app.Writer = buf
	app.ErrWriter = buf

	fullArgs := append([]string{"s6cmd"}, args...)
	err := app.Run(fullArgs)
	return buf.String(), err
}

func TestHelpContainsAllFlags(t *testing.T) {
	output, err := runApp("--help")
	require.NoError(t, err)

	assert.Contains(t, output, "--profile")
	assert.Contains(t, output, "--endpoint-url")
	assert.Contains(t, output, "--region")
	assert.Contains(t, output, "--no-sign-request")
}

func TestPresignHelpContainsFlags(t *testing.T) {
	output, err := runApp("--no-sign-request", "presign", "--help")
	require.NoError(t, err)

	assert.Contains(t, output, "--method")
	assert.Contains(t, output, "--expires-in")
}

func TestPresignMissingArgument(t *testing.T) {
	_, err := runApp("--no-sign-request", "presign")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing S3 URI")
}

func TestPresignInvalidMethod(t *testing.T) {
	_, err := runApp("--no-sign-request", "presign", "--method", "PATCH", "s3://bucket/key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid method")
}

func TestPresignInvalidS3URI(t *testing.T) {
	_, err := runApp("--no-sign-request", "presign", "not-an-s3-uri")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must start with s3://")
}

func TestPresignValidInvocationWithEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")

	output, err := runApp(
		"--endpoint-url", server.URL,
		"--region", "us-east-1",
		"presign",
		"s3://test-bucket/test-key",
	)
	require.NoError(t, err)
	assert.Contains(t, output, "test-bucket")
	assert.Contains(t, output, "test-key")
	assert.Contains(t, output, "X-Amz-Signature")
}

func TestPresignPUTMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")

	output, err := runApp(
		"--endpoint-url", server.URL,
		"--region", "us-east-1",
		"presign",
		"--method", "PUT",
		"s3://test-bucket/upload/file.txt",
	)
	require.NoError(t, err)
	assert.Contains(t, output, "test-bucket")
	assert.Contains(t, output, "upload/file.txt")
	assert.Contains(t, output, "X-Amz-Signature")
}

func TestS5cmdCommandsAvailable(t *testing.T) {
	output, err := runApp("--help")
	require.NoError(t, err)

	// Verify key s5cmd commands are listed
	assert.Contains(t, output, "ls")
	assert.Contains(t, output, "cp")
	assert.Contains(t, output, "mv")
	assert.Contains(t, output, "rm")
	assert.Contains(t, output, "sync")
	assert.Contains(t, output, "presign")
}

func TestPresignExpiresInExceedsMax(t *testing.T) {
	_, err := runApp("--no-sign-request", "presign", "--expires-in", "169h", "s3://b/k")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds AWS maximum")
}

func TestPresignExpiresInZero(t *testing.T) {
	_, err := runApp("--no-sign-request", "presign", "--expires-in", "0s", "s3://b/k")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestNoSignRequestSkipsCredentials(t *testing.T) {
	// Unset any AWS credentials to ensure we're testing the skip path
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/dev/null")
	t.Setenv("AWS_CONFIG_FILE", "/dev/null")

	// --no-sign-request should skip credential resolution entirely
	_, err := runApp("--no-sign-request", "presign", "s3://bucket/key")
	// Presign itself will fail (no creds for signing) but the Before hook should not error
	// The error should be about signing, not about credential resolution
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "failed to load AWS config")
}
