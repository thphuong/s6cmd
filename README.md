# s6cmd

A drop-in wrapper around [s5cmd](https://github.com/peak/s5cmd) with some minor modifications.

- **Reads default credential files** (`~/.aws/credentials`, `~/.aws/config`, SSO, IMDS) properly via AWS SDK v2
- **Presign supports upload** (PUT and DELETE presigned URLs)
- **All s5cmd commands included**

## Installation

```bash
go install github.com/thphuong/s6cmd@latest
```

Or build locally:

```bash
git clone https://github.com/thphuong/s6cmd.git
cd s6cmd
make build
```

## Usage

s6cmd is a full s5cmd replacement. All s5cmd commands work exactly the same:

```bash
s6cmd ls s3://my-bucket/
s6cmd cp file.txt s3://my-bucket/
s6cmd sync /local/dir/ s3://my-bucket/prefix/
s6cmd mv s3://bucket/old-key s3://bucket/new-key
s6cmd rm s3://bucket/key
```

### Presign (enhanced)

Generate presigned GET URL (default 1h expiry):

```bash
s6cmd presign s3://my-bucket/path/to/object
```

Generate presigned PUT URL for upload:

```bash
s6cmd presign s3://my-bucket/uploads/file.txt --method PUT --expires-in 30m
```

With AWS profile and custom endpoint:

```bash
s6cmd --profile my-profile --region us-west-2 presign s3://bucket/key
s6cmd --endpoint-url http://localhost:9000 presign s3://bucket/key
```

### How Credential Resolution Works

Before any command, s6cmd resolves credentials via AWS SDK v2's default chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, etc.)
2. Shared credentials file (`~/.aws/credentials`)
3. Shared config file (`~/.aws/config`)
4. SSO / IAM Identity Center
5. EC2 instance metadata (IMDS)

Resolved credentials are passed to s5cmd commands via environment variables, fixing s5cmd's native credential reading issues.

### Global Flags

| Flag                 | Default | Description                                 |
|----------------------|---------|---------------------------------------------|
| `--profile`          | -       | AWS profile name                            |
| `--region`           | -       | AWS region override                         |
| `--endpoint-url`     | -       | Custom S3 endpoint (MinIO, LocalStack)      |
| `--no-sign-request`  | false   | Skip credential resolution (public buckets) |
| `--credentials-file` | -       | Custom credentials file path                |
| `--numworkers`       | 256     | Parallel workers for s5cmd operations       |
| `--retry-count`      | 10      | Retry count for failed requests             |
| `--json`             | false   | JSON formatted output                       |
| `--dry-run`, `-n`    | false   | Show commands without executing             |
| `--log`              | info    | Log level: trace, debug, info, error        |

### Presign Flags

| Flag           | Default | Description                       |
|----------------|---------|-----------------------------------|
| `--method`     | `GET`   | HTTP method: GET, PUT, or DELETE  |
| `--expires-in` | `1h`    | URL expiration (max: 168h/7 days) |

## Project Structure

```
s6cmd/
├── main.go                          # Entry point
├── cmd/
│   ├── app.go                       # CLI app builder (urfave/cli)
│   ├── app_test.go                  # CLI integration tests
│   └── presign.go                   # Presign command (GET + PUT)
├── internal/
│   ├── presign.go                   # Presigned URL generation (AWS SDK v2)
│   ├── presign_test.go              # Presign tests
│   ├── credential_resolver.go       # AWS credential resolution + env export
│   ├── parse_s3_uri.go              # S3 URI parser
│   └── parse_s3_uri_test.go         # URI parsing tests
├── e2e_test.sh                      # E2E test script for any S3 provider
├── go.mod
├── go.sum
└── Makefile
```

## Testing

### Unit tests

```bash
make test
# or: go test ./...
```

24 tests covering URI parsing, presign logic, CLI integration, credential skipping.

### E2E tests

Run against any S3-compatible provider (AWS, R2, MinIO, LocalStack):

```bash
./e2e_test.sh --profile <aws-profile>
```

Skip tests unsupported by your provider (e.g. bucket create/delete):

```bash
./e2e_test.sh --profile aws-prod --skip mb,rb
```

Options:

| Flag        | Default          | Description                        |
|-------------|------------------|------------------------------------|
| `--profile` | (required)       | AWS profile to use                 |
| `--bucket`  | `s6cmd-e2e-test` | Test bucket name                   |
| `--skip`    | (none)           | Comma-separated test names to skip |
| `--binary`  | `./s6cmd`        | Path to s6cmd binary               |

Skippable tests: `mb`, `rb`, `cp-upload`, `cp-download`, `ls-buckets`, `ls-objects`,
`cat`, `mv`, `head`, `du`, `rm`, `presign-get`, `presign-put`, `presign-del`, `version`.

> **Note:** Skipping `mb` assumes the bucket already exists. Skipping `rm`/`rb` leaves
> the test bucket and objects behind — clean up manually or re-run without `--skip`.

## License

MIT
