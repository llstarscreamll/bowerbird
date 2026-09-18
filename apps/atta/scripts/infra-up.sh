#!/usr/bin/env bash
set -euo pipefail

ATTA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ATTA_DIR}"
docker compose up -d --wait --wait-timeout 180 --remove-orphans
docker compose --profile init run --rm --no-TTY minio-init
