#!/usr/bin/env sh
set -eu

ENDPOINT="${MINIO_ENDPOINT:-http://minio:9000}"
ROOT_USER="${MINIO_ROOT_USER:-bowerbird}"
ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-bowerbirdsecret}"
BUCKET="${MINIO_BUCKET:-bowerbird-local-bucket}"

mc alias set local "$ENDPOINT" "$ROOT_USER" "$ROOT_PASSWORD"
mc mb "local/${BUCKET}" --ignore-existing

echo "MinIO bucket ready: ${BUCKET}"
