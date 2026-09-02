# MinIO

S3-compatible object storage for local development and on-prem deployments.

## Local development

- API: `http://localhost:9000`
- Console: `http://localhost:9001` (credentials below)
- Bootstrap: `apps/backend/scripts/init-minio.sh` (runs via `minio-init` on `pnpm run infra:up`)
- Backend: `MINIO_ENDPOINT_URL=http://localhost:9000`
- Browser uploads/downloads: `S3_PRESIGN_ENDPOINT_URL=https://media.bowerbird.dev` (Caddy → MinIO `:9000`)

Default credentials (root `docker-compose.yml`):

| Variable                                        | Value                    |
| ----------------------------------------------- | ------------------------ |
| `AWS_ACCESS_KEY_ID` / `MINIO_ROOT_USER`         | `bowerbird`              |
| `AWS_SECRET_ACCESS_KEY` / `MINIO_ROOT_PASSWORD` | `bowerbirdsecret`        |
| `S3_BUCKET_NAME`                                | `bowerbird-local-bucket` |

`infra:up` creates the bucket. Browser CORS is configured on the MinIO service via `MINIO_API_CORS_ALLOW_ORIGIN`.

If uploads fail with `SignatureDoesNotMatch`:

1. Confirm `.env` has `AWS_REQUEST_CHECKSUM_CALCULATION=when_required` and restart workers.
2. Check for **non-ASCII characters in S3 user metadata** (e.g. macOS screenshot filenames with narrow no-break spaces). The object store adapter sanitizes metadata automatically — see [Object storage](../architecture/object-storage.md).
3. Confirm startup log shows `object storage ready: endpoint=http://localhost:9000`.

Re-run bucket bootstrap manually:

```bash
docker compose --profile init run --rm --no-TTY minio-init
```

## On-prem production

See [On-prem stack](../architecture/onprem-runtime.md) (`deploy/onprem/docker-compose.yml`). MinIO runs as an internal service; Caddy terminates HTTPS for API and media.

## AWS profile

Production uses real S3 (no MinIO). Set `DEPLOYMENT_TARGET=aws` and load secrets from SSM. See [AWS deploy](../deployment/aws.md).
