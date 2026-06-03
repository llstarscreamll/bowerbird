#!/usr/bin/env bash
set -euo pipefail

REGION="${AWS_DEFAULT_REGION:-us-east-1}"
ACCOUNT_ID="000000000000"

SQS_QUEUE_NAME="bowerbird-local-sqs"
EVENTBRIDGE_QUEUE_NAME="bowerbird-local-eventbridge"
EVENT_BUS_NAME="bowerbird-local-bus"
EVENT_RULE_NAME="bowerbird-local-rule"
S3_BUCKET_NAME="bowerbird-local-bucket"
SSM_PARAMETER_NAME="/bowerbird/local/secrets"

awslocal sqs create-queue --queue-name "$SQS_QUEUE_NAME" >/dev/null
awslocal sqs create-queue --queue-name "$EVENTBRIDGE_QUEUE_NAME" >/dev/null

EVENTBRIDGE_QUEUE_ARN="arn:aws:sqs:${REGION}:${ACCOUNT_ID}:${EVENTBRIDGE_QUEUE_NAME}"

awslocal events create-event-bus --name "$EVENT_BUS_NAME" >/dev/null || true
awslocal events put-rule \
  --name "$EVENT_RULE_NAME" \
  --event-bus-name "$EVENT_BUS_NAME" \
  --event-pattern '{"source": [{"prefix": "bowerbird."}]}' >/dev/null
awslocal events put-targets \
  --event-bus-name "$EVENT_BUS_NAME" \
  --rule "$EVENT_RULE_NAME" \
  --targets "Id"="eventbridge-queue","Arn"="$EVENTBRIDGE_QUEUE_ARN" >/dev/null

awslocal s3api create-bucket --bucket "$S3_BUCKET_NAME" >/dev/null || true

awslocal s3api put-bucket-cors \
  --bucket "$S3_BUCKET_NAME" \
  --cors-configuration '{
    "CORSRules": [
      {
        "AllowedOrigins": [
          "https://app.bowerbird.dev",
          "http://localhost:4200"
        ],
        "AllowedMethods": ["GET", "PUT", "HEAD"],
        "AllowedHeaders": ["content-type", "x-amz-*"],
        "ExposeHeaders": ["ETag", "x-amz-request-id", "x-amz-id-2"],
        "MaxAgeSeconds": 300
      }
    ]
  }' >/dev/null

# Configurar secretos en un SecureString
SECRETS_FILE="/tmp/local-secrets.json"
if [ -f "$SECRETS_FILE" ]; then
  SECRETS_JSON=$(cat "$SECRETS_FILE")
  echo "Cargando secretos desde $SECRETS_FILE"
else
  # Fallback si no hay archivo
  echo "Archivo $SECRETS_FILE no encontrado, usando secretos dummy por defecto"
  SECRETS_JSON=$(cat <<EOF
{
  "database_url": "postgres://bowerbird:bowerbird@postgres:5432/bowerbird?sslmode=disable",
  "sqs_queue_url": "http://localhost:4566/000000000000/bowerbird-local-sqs",
  "eventbridge_queue_url": "http://localhost:4566/000000000000/bowerbird-local-eventbridge",
  "event_bus_name": "bowerbird-local-bus",
  "s3_bucket_name": "bowerbird-local-bucket",
  "third_party_api_key": "dummy-api-key",
  "google_client_id": "dummy-google-client-id",
  "google_client_secret": "dummy-google-client-secret",
  "microsoft_client_id": "dummy-microsoft-client-id",
  "microsoft_client_secret": "dummy-microsoft-client-secret"
}
EOF
)
fi

awslocal ssm put-parameter \
  --name "$SSM_PARAMETER_NAME" \
  --type "SecureString" \
  --value "$SECRETS_JSON" \
  --overwrite >/dev/null

echo "LocalStack resources created: SQS, EventBridge, S3 (with CORS), SSM"
