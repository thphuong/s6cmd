package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/peak/s5cmd/v2/command"
	"github.com/peak/s5cmd/v2/log"
	"github.com/peak/s5cmd/v2/log/stat"
	"github.com/peak/s5cmd/v2/parallel"
	"github.com/thphuong/s6cmd/internal"
	"github.com/urfave/cli/v2"
)

// NewApp builds the s6cmd CLI application. It imports all s5cmd commands,
// replaces presign with our enhanced version (GET + PUT), and adds a
// Before hook that resolves AWS credentials via SDK v2.
func NewApp() *cli.App {
	cmds := command.Commands()

	// Replace s5cmd's presign with ours
	var filtered []*cli.Command
	for _, c := range cmds {
		if c.Name != "presign" {
			filtered = append(filtered, c)
		}
	}
	filtered = append(filtered, newPresignCommand())

	return &cli.App{
		Name:     "s6cmd",
		Usage:    "S3 file operations",
		Commands: filtered,
		Flags:    globalFlags(),
		Before:   beforeHook,
		After:    afterHook,
		Action: func(c *cli.Context) error {
			args := c.Args()
			if args.Present() {
				cli.ShowCommandHelp(c, args.First())
				return cli.Exit("", 1)
			}
			return cli.ShowAppHelp(c)
		},
	}
}

// globalFlags returns s5cmd's global flags plus --region.
func globalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "json",
			Usage: "enable JSON formatted output",
		},
		&cli.IntFlag{
			Name:  "numworkers",
			Value: 256,
			Usage: "number of workers execute operation on each object",
		},
		&cli.IntFlag{
			Name:    "retry-count",
			Aliases: []string{"r"},
			Value:   10,
			Usage:   "number of times that a request will be retried for failures",
		},
		&cli.StringFlag{
			Name:    "endpoint-url",
			Usage:   "override default S3 host for custom services",
			EnvVars: []string{"S3_ENDPOINT_URL"},
		},
		&cli.BoolFlag{
			Name:  "no-verify-ssl",
			Usage: "disable SSL certificate verification",
		},
		&cli.GenericFlag{
			Name: "log",
			Value: &command.EnumValue{
				Enum:    []string{"trace", "debug", "info", "error"},
				Default: "info",
			},
			Usage: "log level: (trace, debug, info, error)",
		},
		&cli.BoolFlag{
			Name:    "dry-run",
			Aliases: []string{"n"},
			Usage:   "fake run; show what commands will be executed without actually executing them",
		},
		&cli.BoolFlag{
			Name:  "stat",
			Usage: "collect statistics of program execution and display it at the end",
		},
		&cli.BoolFlag{
			Name:  "no-sign-request",
			Usage: "do not sign requests: credentials will not be loaded if --no-sign-request is provided",
		},
		&cli.BoolFlag{
			Name:  "use-list-objects-v1",
			Usage: "use ListObjectsV1 API for services that don't support ListObjectsV2",
		},
		&cli.StringFlag{
			Name:  "request-payer",
			Usage: "who pays for request (access requester pays buckets)",
		},
		&cli.StringFlag{
			Name:  "profile",
			Usage: "use the specified profile from the credentials file",
		},
		&cli.StringFlag{
			Name:  "credentials-file",
			Usage: "use the specified credentials file instead of the default credentials file",
		},
		// s6cmd additions
		&cli.StringFlag{
			Name:  "region",
			Usage: "AWS region override",
		},
	}
}

// commandsSkippingCreds lists commands that should work without AWS credentials.
var commandsSkippingCreds = map[string]bool{
	"version": true,
	"help":    true,
}

// beforeHook runs before any command. It initializes s5cmd internals
// (logging, parallel workers) and resolves AWS credentials via SDK v2.
func beforeHook(c *cli.Context) error {
	// Initialize s5cmd internals
	log.Init(c.String("log"), c.Bool("json"))
	parallel.Init(c.Int("numworkers"))

	if c.Int("retry-count") < 0 {
		return fmt.Errorf("retry count cannot be a negative value")
	}
	if c.Bool("no-sign-request") && c.String("profile") != "" {
		return fmt.Errorf(`"no-sign-request" and "profile" flags cannot be used together`)
	}
	if c.Bool("no-sign-request") && c.String("credentials-file") != "" {
		return fmt.Errorf(`"no-sign-request" and "credentials-file" flags cannot be used together`)
	}

	if c.Bool("stat") {
		stat.InitStat()
	}

	endpointURL := c.String("endpoint-url")
	if endpointURL != "" && !strings.HasPrefix(endpointURL, "http") {
		return fmt.Errorf(`bad value for --endpoint-url %v: scheme is missing. Must be of the form http://<hostname>/ or https://<hostname>/`, endpointURL)
	}

	// Skip credential resolution for no-sign-request, help, or version
	if c.Bool("no-sign-request") {
		return nil
	}
	cmdName := c.Args().First()
	if commandsSkippingCreds[cmdName] || cmdName == "" {
		return nil
	}

	// Resolve credentials via AWS SDK v2 and export as env vars for s5cmd
	resolved, err := internal.ResolveAndSetCredentials(c.Context, c.String("profile"), c.String("region"))
	if err != nil {
		return err
	}

	// Clear profile/credentials-file so s5cmd's storage layer uses env vars
	// instead of SDK v1 SharedCredentials (which doesn't support SSO/assume-role).
	c.Set("profile", "")
	c.Set("credentials-file", "")

	// If the profile had a custom endpoint and none was provided via flag,
	// propagate it so s5cmd connects to the right service (e.g. R2, MinIO).
	if c.String("endpoint-url") == "" && resolved.BaseEndpoint != "" {
		c.Set("endpoint-url", resolved.BaseEndpoint)
	}

	return nil
}

// afterHook cleans up s5cmd internals.
func afterHook(c *cli.Context) error {
	if c.Bool("stat") && len(stat.Statistics()) > 0 {
		log.Stat(stat.Statistics())
	}
	// s5cmd uses package-level channels that panic on double-close in tests
	defer func() {
		if r := recover(); r != nil {
			_ = r // expected: s5cmd channel double-close
		}
	}()
	parallel.Close()
	log.Close()
	return nil
}

// Execute runs the s6cmd application.
func Execute() {
	app := NewApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
