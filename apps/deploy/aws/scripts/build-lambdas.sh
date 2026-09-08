#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
BACKEND="$ROOT/apps/backend"
OUT="$ROOT/apps/deploy/aws/.build/lambda"

build_one() {
  local name="$1"
  local dir="$OUT/$name"
  mkdir -p "$dir"
  (
    cd "$BACKEND"
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -ldflags="-s -w" -o "$dir/bootstrap" "./cmd/aws/lambda/$name"
  )
}

build_one http
build_one sqs
build_one eventbridge
build_one outbox-relay
build_one scheduler
build_one migrate

cp -a "$BACKEND/migrations" "$OUT/http/migrations"
cp -a "$BACKEND/migrations" "$OUT/migrate/migrations"
