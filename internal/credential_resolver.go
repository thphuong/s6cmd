package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ResolvedConfig holds resolved AWS configuration values for the caller.
type ResolvedConfig struct {
	BaseEndpoint string
}

// ResolveAndSetCredentials loads AWS credentials via SDK v2 default chain
// and sets AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN,
// and AWS_DEFAULT_REGION as environment variables so s5cmd commands can use them.
// It returns the resolved config so the caller can also set endpoint-url, etc.
func ResolveAndSetCredentials(ctx context.Context, profile, region string) (*ResolvedConfig, error) {
	var opts []func(*config.LoadOptions) error
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	os.Setenv("AWS_ACCESS_KEY_ID", creds.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey)
	if creds.SessionToken != "" {
		os.Setenv("AWS_SESSION_TOKEN", creds.SessionToken)
	} else {
		os.Unsetenv("AWS_SESSION_TOKEN")
	}

	if region != "" {
		os.Setenv("AWS_DEFAULT_REGION", region)
	} else if cfg.Region != "" {
		os.Setenv("AWS_DEFAULT_REGION", cfg.Region)
	}

	resolved := &ResolvedConfig{}
	if cfg.BaseEndpoint != nil {
		resolved.BaseEndpoint = *cfg.BaseEndpoint
	} else {
		// BaseEndpoint doesn't capture service-specific endpoints (e.g. from
		// [services ...] sections). Create an S3 client to resolve the endpoint.
		s3Client := s3.NewFromConfig(cfg)
		if ep := s3Client.Options().BaseEndpoint; ep != nil {
			resolved.BaseEndpoint = *ep
		}
	}

	return resolved, nil
}
