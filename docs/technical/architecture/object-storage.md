# Object storage

All file I/O goes through the `FileStore` port (`internal/platform/storage/s3`). The same adapter serves **AWS S3** (`DEPLOYMENT_TARGET=aws`) and **MinIO** (`DEPLOYMENT_TARGET=onprem`).

## Configuration

| Variable                                      | Purpose                                                                         |
| --------------------------------------------- | ------------------------------------------------------------------------------- |
| `S3_BUCKET_NAME`                              | Target bucket                                                                   |
| `AWS_REGION`                                  | SDK region (required even for MinIO)                                            |
| `MINIO_ENDPOINT_URL`                          | Custom S3 endpoint (onprem only; omit on AWS)                                   |
| `S3_PRESIGN_ENDPOINT_URL`                     | Public URL for browser presigned uploads (local: `https://media.bowerbird.dev`) |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | Credentials                                                                     |
| `AWS_REQUEST_CHECKSUM_CALCULATION`            | Set `when_required` for MinIO compatibility                                     |
| `AWS_RESPONSE_CHECKSUM_VALIDATION`            | Set `when_required` for MinIO compatibility                                     |

See [MinIO](../tooling/minio.md) for local bootstrap and credentials.

## Metadata sanitization (adapter boundary)

S3 user metadata values must be **US-ASCII**. Non-ASCII characters in metadata headers (e.g. Unicode narrow no-break space `U+202F` in macOS screenshot filenames like `Screenshot … at 8.12.01 AM.png`) cause `PutObject` to fail with `SignatureDoesNotMatch` on MinIO and can fail on AWS S3 as well.

The object store adapter sanitizes metadata on every write:

- `PutObject` — `internal/platform/storage/s3/object_store.go`
- `PresignUpload` — same adapter

Implementation: `SanitizeObjectMetadata()` / `SanitizeObjectMetadataValue()` in `internal/platform/storage/metadata.go` (non-ASCII → ASCII spaces, control chars stripped).

**Domain and DB keep the original filename.** Only the S3 `orig_name` metadata header is normalized at the storage adapter. Callers pass raw filenames; do not duplicate sanitization in feature modules.

## Operations

| Method                              | Use                                       |
| ----------------------------------- | ----------------------------------------- |
| `WriteFileIfAbsent`                 | Inbox attachment sync (idempotent by key) |
| `PresignUpload` / `PresignDownload` | Browser file uploads (Files module)       |
| `HeadObject`                        | Existence checks                          |

Object keys follow `internal/platform/storage/object_key.go` conventions (tenant-scoped paths).

## Troubleshooting

| Symptom                                           | Likely cause                                                                                      |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `SignatureDoesNotMatch` on `PutObject` (MinIO)    | Non-ASCII in user metadata **or** stale worker after SDK/checksum config change — restart workers |
| Presigned URL works from CLI but fails in browser | `S3_PRESIGN_ENDPOINT_URL` / Caddy `media.*` routing                                               |
| Upload 403 CORS                                   | MinIO `MINIO_API_CORS_ALLOW_ORIGIN` must include the PWA origin                                   |

Integration test reproducing macOS filename metadata: `TestMinIOMacOSScreenshotFilenameMetadata` in `internal/platform/storage/s3/minio_integration_test.go`.
