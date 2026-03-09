package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseS3URI(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantBucket string
		wantKey    string
		wantErr    bool
		errContain string
	}{
		{
			name:       "valid simple URI",
			input:      "s3://mybucket/mykey",
			wantBucket: "mybucket",
			wantKey:    "mykey",
		},
		{
			name:       "valid nested key",
			input:      "s3://mybucket/path/to/file.txt",
			wantBucket: "mybucket",
			wantKey:    "path/to/file.txt",
		},
		{
			name:       "empty key (trailing slash only)",
			input:      "s3://mybucket/",
			wantErr:    true,
			errContain: "missing object key",
		},
		{
			name:       "no key (no slash)",
			input:      "s3://mybucket",
			wantErr:    true,
			errContain: "missing object key",
		},
		{
			name:       "missing s3:// prefix",
			input:      "mybucket/mykey",
			wantErr:    true,
			errContain: "must start with s3://",
		},
		{
			name:       "empty string",
			input:      "",
			wantErr:    true,
			errContain: "empty",
		},
		{
			name:       "s3:// only (no bucket)",
			input:      "s3://",
			wantErr:    true,
			errContain: "missing bucket",
		},
		{
			name:       "wrong scheme (https)",
			input:      "https://mybucket/key",
			wantErr:    true,
			errContain: "must start with s3://",
		},
		{
			name:       "key with spaces",
			input:      "s3://mybucket/my file.txt",
			wantBucket: "mybucket",
			wantKey:    "my file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := ParseS3URI(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantBucket, bucket)
			assert.Equal(t, tt.wantKey, key)
		})
	}
}
