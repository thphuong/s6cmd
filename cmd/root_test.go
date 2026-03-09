package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// executeCommand creates a fresh root command and runs it with given args.
func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd := NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return buf.String(), err
}

func TestHelpContainsAllFlags(t *testing.T) {
	output, err := executeCommand("--help")
	require.NoError(t, err)

	assert.Contains(t, output, "--profile")
	assert.Contains(t, output, "--method")
	assert.Contains(t, output, "--expires-in")
	assert.Contains(t, output, "--endpoint-url")
	assert.Contains(t, output, "--region")
}

func TestMissingArgument(t *testing.T) {
	_, err := executeCommand()
	require.Error(t, err)
}

func TestInvalidMethod(t *testing.T) {
	_, err := executeCommand("--method", "DELETE", "s3://bucket/key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid method")
}

func TestInvalidS3URI(t *testing.T) {
	_, err := executeCommand("not-an-s3-uri")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must start with s3://")
}

func TestValidInvocationWithEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")

	output, err := executeCommand(
		"--endpoint-url", server.URL,
		"--region", "us-east-1",
		"s3://test-bucket/test-key",
	)
	require.NoError(t, err)
	assert.Contains(t, output, "test-bucket")
	assert.Contains(t, output, "test-key")
	assert.Contains(t, output, "X-Amz-Signature")
}
