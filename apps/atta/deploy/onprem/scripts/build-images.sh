#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../../.." && pwd)"
cd "$ROOT"

ONPREM_RELEASE="${ONPREM_RELEASE:?ONPREM_RELEASE is required (image tag / git sha / version)}"
export ONPREM_RELEASE

pnpm --filter @atta/pwa build
docker compose -f apps/atta/deploy/onprem/docker-compose.yml build
