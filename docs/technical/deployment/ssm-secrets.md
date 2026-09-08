# AWS secrets (Secrets Manager)

Applies when `DEPLOYMENT_TARGET=aws`. Local and on-prem use plain `.env`
([backend-api](../architecture/backend-api.md#config-and-secrets)).

AWS Lambdas load one JSON secret at boot (`config.Load()`). The primary store
is **AWS Secrets Manager**. SSM Parameter Store SecureString remains a
fallback for existing parameters.

## Parameter

| Setting                                    | Default | Description                          |
| ------------------------------------------ | ------- | ------------------------------------ |
| `SECRET_ARN` / `SECRETS_MANAGER_SECRET_ID` | (none)  | Secrets Manager name or ARN          |
| `SSM_PARAMETER_NAME`                       | (none)  | Used only when `SECRET_ARN` is empty |

Pulumi writes `/bowerbird/${ENV}/app` and sets `SECRET_ARN` on every Lambda.
Keys use **snake_case** JSON field names matching struct tags in
`apps/backend/internal/platform/config/config.go`.

Do not put this JSON in Lambda environment variables. Env holds non-secret
routing only (`DEPLOYMENT_TARGET`, `SECRET_ARN`, public URLs).

## Required fields (AWS)

| Key                                | Type   | Description                                            |
| ---------------------------------- | ------ | ------------------------------------------------------ |
| `database_url`                     | string | Neon **pooled** Postgres URL (Lambda request path)     |
| `database_direct_url`              | string | Neon **direct** URL (migrations and `CREATE DATABASE`) |
| `sqs_queue_url`                    | string | SQS queue URL for background jobs                      |
| `event_bus_name`                   | string | EventBridge custom event bus name                      |
| `s3_bucket_name`                   | string | S3 bucket for object storage                           |
| `inbox_credentials_encryption_key` | string | Base64-encoded 32-byte AES key for inbox OAuth tokens  |
| `tenant_secrets_encryption_key`    | string | Base64-encoded key for tenant document passwords       |
| `gemini_api_key`                   | string | Google Gemini API key (invoice extraction)             |
| `jwt_access_secret`                | string | JWT access-token signing secret                        |
| `jwt_refresh_secret`               | string | JWT refresh-token signing secret                       |
| `messaging_attestation_secret`     | string | HMAC secret for job/event tenant attestation           |

`eventbridge_queue_url` is optional. The AWS events Lambda is invoked by
EventBridge directly.

## Optional fields (merged when present)

| Key                       | Type    | Description                                  |
| ------------------------- | ------- | -------------------------------------------- |
| `google_client_id`        | string  | Gmail OAuth client ID                        |
| `google_client_secret`    | string  | Gmail OAuth client secret                    |
| `microsoft_client_id`     | string  | Microsoft mail OAuth client ID               |
| `microsoft_client_secret` | string  | Microsoft mail OAuth client secret           |
| `gemini_model`            | string  | Gemini model id (default `gemini-2.0-flash`) |
| `gemini_endpoint`         | string  | Gemini API base URL                          |
| `app_env`                 | string  | Runtime environment label                    |
| `allowed_origins`         | string  | Comma-separated CORS origins                 |
| `frontend_url`            | string  | PWA base URL                                 |
| `backend_url`             | string  | API base URL                                 |
| `debug`                   | boolean | Enable debug mode                            |

## Not in the secret (Lambda env only)

- `DEPLOYMENT_TARGET=aws`
- `AWS_REGION`
- `SECRET_ARN`
- `TENANT_MIGRATIONS_DIR` (HTTP and migrate Lambdas)

On-prem-only keys (`rabbitmq_url`, `minio_endpoint_url`, …) belong in `.env`,
not this secret.

## Neon URLs

Use the pooler hostname (`-pooler`) for `database_url`. PgBouncer transaction
mode cannot run `CREATE DATABASE` or some migration session features, so
`database_direct_url` must omit `-pooler`.

Pulumi fills both from the Neon project outputs.

## Example payload

```json
{
  "database_url": "postgres://bowerbird:secret@ep-xxx-pooler.us-east-1.aws.neon.tech/bowerbird?sslmode=require",
  "database_direct_url": "postgres://bowerbird:secret@ep-xxx.us-east-1.aws.neon.tech/bowerbird?sslmode=require",
  "sqs_queue_url": "https://sqs.us-east-1.amazonaws.com/ACCOUNT_ID/prod-bowerbird-jobs",
  "event_bus_name": "prod-bowerbird-bus",
  "s3_bucket_name": "prod-bowerbird-ACCOUNT-objects",
  "google_client_id": "your-google-client-id.apps.googleusercontent.com",
  "google_client_secret": "your-google-client-secret",
  "microsoft_client_id": "your-microsoft-client-id",
  "microsoft_client_secret": "your-microsoft-client-secret",
  "gemini_api_key": "your-gemini-api-key",
  "gemini_model": "gemini-2.0-flash",
  "gemini_endpoint": "https://generativelanguage.googleapis.com",
  "inbox_credentials_encryption_key": "base64-encoded-32-byte-key",
  "tenant_secrets_encryption_key": "base64-encoded-32-byte-key",
  "jwt_access_secret": "generated-access-secret",
  "jwt_refresh_secret": "generated-refresh-secret",
  "messaging_attestation_secret": "generated-attestation-secret"
}
```

After updating the secret, deploy new Lambda versions or wait for cold starts
so processes reload config.
