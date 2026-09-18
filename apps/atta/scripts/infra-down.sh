#!/usr/bin/env bash
set -euo pipefail

ATTA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ATTA_DIR}"
docker compose down
