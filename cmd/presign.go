package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/thphuong/s6cmd/internal"
	"github.com/urfave/cli/v2"
)

const maxPresignExpiry = 7 * 24 * time.Hour

func newPresignCommand() *cli.Command {
	return &cli.Command{
		Name:  "presign",
		Usage: "generate presigned URL",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "method",
				Value: "GET",
				Usage: "HTTP method: GET, PUT, or DELETE",
			},
			&cli.DurationFlag{
				Name:  "expires-in",
				Value: time.Hour,
				Usage: "URL expiration (max 168h / 7 days)",
			},
		},
		Action: presignAction,
	}
}

func presignAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("missing S3 URI argument")
	}

	method := strings.ToUpper(c.String("method"))
	if method != "GET" && method != "PUT" && method != "DELETE" {
		return fmt.Errorf("invalid method %q: must be GET, PUT, or DELETE", method)
	}

	expiresIn := c.Duration("expires-in")
	if expiresIn <= 0 {
		return fmt.Errorf("--expires-in must be positive")
	}
	if expiresIn > maxPresignExpiry {
		return fmt.Errorf("--expires-in %s exceeds AWS maximum of 7 days (168h)", expiresIn)
	}

	bucket, key, err := internal.ParseS3URI(c.Args().First())
	if err != nil {
		return err
	}

	opts := internal.PresignOptions{
		Bucket:      bucket,
		Key:         key,
		Method:      method,
		ExpiresIn:   expiresIn,
		Profile:     c.String("profile"),
		Region:      c.String("region"),
		EndpointURL: c.String("endpoint-url"),
	}

	url, err := internal.GeneratePresignedURL(c.Context, opts)
	if err != nil {
		return err
	}

	fmt.Fprintln(c.App.Writer, url)
	return nil
}
