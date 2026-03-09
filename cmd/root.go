package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thphuong/s6cmd/internal"
)

// NewRootCmd creates and returns the root cobra command with all flags configured.
func NewRootCmd() *cobra.Command {
	var (
		profile     string
		method      string
		expiresIn   time.Duration
		endpointURL string
		region      string
	)

	cmd := &cobra.Command{
		Use:          "s6cmd presign s3://<bucket>/<key>",
		Short:        "s6cmd - s5cmd wrapper with credential fixes and presign upload support",
		Long:         "s6cmd wraps s5cmd and fixes annoying issues like not reading default credential files and presign not supporting upload.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := strings.ToUpper(method)
			if m != "GET" && m != "PUT" {
				return fmt.Errorf("invalid method %q: must be GET or PUT", method)
			}

			// AWS caps presigned URLs at 7 days
			const maxExpiry = 7 * 24 * time.Hour
			if expiresIn > maxExpiry {
				return fmt.Errorf("--expires-in %s exceeds AWS maximum of 7 days (168h)", expiresIn)
			}
			if expiresIn <= 0 {
				return fmt.Errorf("--expires-in must be positive")
			}

			bucket, key, err := internal.ParseS3URI(args[0])
			if err != nil {
				return err
			}

			opts := internal.PresignOptions{
				Bucket:      bucket,
				Key:         key,
				Method:      m,
				ExpiresIn:   expiresIn,
				Profile:     profile,
				Region:      region,
				EndpointURL: endpointURL,
			}

			url, err := internal.GeneratePresignedURL(context.Background(), opts)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), url)
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "AWS profile name")
	cmd.Flags().StringVar(&method, "method", "GET", "HTTP method: GET or PUT")
	cmd.Flags().DurationVar(&expiresIn, "expires-in", 3600*time.Second, "URL expiration time (e.g. 1h, 900s)")
	cmd.Flags().StringVar(&endpointURL, "endpoint-url", "", "Custom S3 endpoint URL")
	cmd.Flags().StringVar(&region, "region", "", "AWS region")

	return cmd
}

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
