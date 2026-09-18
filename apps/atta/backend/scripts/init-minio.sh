#!/usr/bin/env sh
set -eu

ENDPOINT="${MINIO_ENDPOINT:-http://minio:9000}"
ROOT_USER="${MINIO_ROOT_USER:-atta}"
ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-attasecret}"
BUCKET="${MINIO_BUCKET:-atta-local-bucket}"

mc alias set local "$ENDPOINT" "$ROOT_USER" "$ROOT_PASSWORD"
mc mb "local/${BUCKET}" --ignore-existing

echo "MinIO bucket ready: ${BUCKET}"
