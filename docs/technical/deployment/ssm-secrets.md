# SSM secrets JSON (AWS profile)

Applies only when `DEPLOYMENT_TARGET=aws`. Local and on-prem use plain `.env` ([backend-api](../architecture/backend-api.md#config-and-secrets)).

## Parameter

| Setting              | Default                    | Description                             |
| -------------------- | -------------------------- | --------------------------------------- |
| `SSM_PARAMETER_NAME` | `/bowerbird/local/secrets` | SSM Parameter Store path (SecureString) |

At boot, `config.Load()` fetches the parameter and unmarshals its value into `Config` (`apps/backend/internal/platform/config/config.go`). Keys use **snake_case** JSON field names matching struct tags.

## Required fields (AWS)

| Key                                | Type   | Description                                                  |
| ---------------------------------- | ------ | ------------------------------------------------------------ |
| `database_url`                     | string | PostgreSQL connection URL (control-plane + tenant routing)   |
| `sqs_queue_url`                    | string | SQS queue URL for background jobs                            |
| `eventbridge_queue_url`            | string | SQS queue URL subscribed to EventBridge (integration events) |
| `event_bus_name`                   | string | EventBridge custom event bus name                            |
| `s3_bucket_name`                   | string | S3 bucket for object storage                                 |
| `inbox_credentials_encryption_key` | string | Base64-encoded 32-byte AES key for inbox OAuth tokens        |
| `tenant_secrets_encryption_key`    | string | Base64-encoded key for tenant document passwords             |
| `gemini_api_key`                   | string | Google Gemini API key (invoice extraction)                   |

## Optional fields (merged when present)

| Key                       | Type    | Description                                                           |
| ------------------------- | ------- | --------------------------------------------------------------------- |
| `google_client_id`        | string  | Gmail OAuth client ID                                                 |
| `google_client_secret`    | string  | Gmail OAuth client secret                                             |
| `microsoft_client_id`     | string  | Microsoft mail OAuth client ID                                        |
| `microsoft_client_secret` | string  | Microsoft mail OAuth client secret                                    |
| `gemini_model`            | string  | Gemini model id (default `gemini-2.0-flash` if unset in env fallback) |
| `gemini_endpoint`         | string  | Gemini API base URL                                                   |
| `app_env`                 | string  | Runtime environment label                                             |
| `port`                    | string  | HTTP listen port                                                      |
| `allowed_origins`         | string  | Comma-separated CORS origins                                          |
| `frontend_url`            | string  | PWA base URL                                                          |
| `backend_url`             | string  | API base URL                                                          |
| `debug`                   | boolean | Enable debug mode                                                     |

## Not in SSM (env-only)

Set on the Lambda/task environment, not inside the JSON blob:

- `DEPLOYMENT_TARGET=aws`
- `AWS_REGION`
- `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` (required in non-local `app_env`)

On-prem-only keys (`rabbitmq_url`, `minio_endpoint_url`, …) belong in `.env`, not this parameter.

## Example payload

```json
{
  "database_url": "postgres://user:pass@host:5432/bowerbird?sslmode=require",
  "sqs_queue_url": "https://sqs.us-east-1.amazonaws.com/ACCOUNT_ID/bowerbird-jobs",
  "eventbridge_queue_url": "https://sqs.us-east-1.amazonaws.com/ACCOUNT_ID/bowerbird-events",
  "event_bus_name": "bowerbird-bus",
  "s3_bucket_name": "bowerbird-assets",
  "google_client_id": "your-google-client-id.apps.googleusercontent.com",
  "google_client_secret": "your-google-client-secret",
  "microsoft_client_id": "your-microsoft-client-id",
  "microsoft_client_secret": "your-microsoft-client-secret",
  "gemini_api_key": "your-gemini-api-key",
  "gemini_model": "gemini-2.0-flash",
  "gemini_endpoint": "https://generativelanguage.googleapis.com",
  "inbox_credentials_encryption_key": "base64-encoded-32-byte-key",
  "tenant_secrets_encryption_key": "base64-encoded-32-byte-key"
}
```

Store as **SecureString**. After updating the parameter, restart API/worker processes so they reload config.
