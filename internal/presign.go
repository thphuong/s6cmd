package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// PresignOptions holds configuration for generating a presigned URL.
type PresignOptions struct {
	Bucket      string
	Key         string
	Method      string // GET, PUT, or DELETE
	ExpiresIn   time.Duration
	Profile     string
	Region      string
	EndpointURL string
}

// GeneratePresignedURL creates a presigned S3 URL for GET or PUT operations.
func GeneratePresignedURL(ctx context.Context, opts PresignOptions) (string, error) {
	// Build config options
	var configOpts []func(*config.LoadOptions) error

	if opts.Profile != "" {
		configOpts = append(configOpts, config.WithSharedConfigProfile(opts.Profile))
	}
	if opts.Region != "" {
		configOpts = append(configOpts, config.WithRegion(opts.Region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Build S3 client options
	var s3Opts []func(*s3.Options)
	if opts.EndpointURL != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(opts.EndpointURL)
		})
	}
	// Custom endpoints typically don't support virtual-host style addressing.
	if opts.EndpointURL != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, s3Opts...)
	presignClient := s3.NewPresignClient(client)

	switch opts.Method {
	case "GET":
		req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(opts.Bucket),
			Key:    aws.String(opts.Key),
		}, s3.WithPresignExpires(opts.ExpiresIn))
		if err != nil {
			return "", fmt.Errorf("failed to presign GET request: %w", err)
		}
		return req.URL, nil

	case "PUT":
		req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(opts.Bucket),
			Key:    aws.String(opts.Key),
		}, s3.WithPresignExpires(opts.ExpiresIn))
		if err != nil {
			return "", fmt.Errorf("failed to presign PUT request: %w", err)
		}
		return req.URL, nil

	case "DELETE":
		req, err := presignClient.PresignDeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(opts.Bucket),
			Key:    aws.String(opts.Key),
		}, s3.WithPresignExpires(opts.ExpiresIn))
		if err != nil {
			return "", fmt.Errorf("failed to presign DELETE request: %w", err)
		}
		return req.URL, nil

	default:
		return "", fmt.Errorf("unsupported method %q: must be GET, PUT, or DELETE", opts.Method)
	}
}
