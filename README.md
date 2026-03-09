# s6cmd

A wrapper around [s5cmd](https://github.com/peak/s5cmd) that fixes common annoyances:

- **Reads default credential files** (`~/.aws/credentials`, `~/.aws/config`) properly
- **Presign supports upload** (PUT presigned URLs, not just GET)

Built with Go, AWS SDK v2, and Cobra.

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

### Presign (current feature)

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
s6cmd presign s3://bucket/key --profile my-profile --region us-west-2
s6cmd presign s3://bucket/key --endpoint-url http://localhost:9000
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--method` | `GET` | HTTP method: GET or PUT |
| `--expires-in` | `1h` | URL expiration (max: 168h/7 days) |
| `--profile` | - | AWS profile name |
| `--region` | - | AWS region |
| `--endpoint-url` | - | Custom S3 endpoint (MinIO, LocalStack) |

### Environment Variables

Standard AWS env vars are supported: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION`, `AWS_ENDPOINT_URL`.

## Project Structure

```
s6cmd/
├── main.go                    # Entry point
├── cmd/
│   ├── root.go               # Root command, presign flags and logic
│   └── root_test.go          # CLI integration tests
├── internal/
│   ├── presign.go            # Presigned URL generation (AWS SDK v2)
│   ├── presign_test.go       # Presign tests
│   ├── parse_s3_uri.go       # S3 URI parser
│   └── parse_s3_uri_test.go  # URI parsing tests (9 cases)
├── go.mod
├── go.sum
└── Makefile
```

## Testing

```bash
make test
# or: go test -v ./...
```

18 tests covering URI parsing, presign logic, CLI flag validation.

## License

MIT
