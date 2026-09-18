#!/usr/bin/env bash
set -euo pipefail

ATTA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_DIR="$(cd "${ATTA_DIR}/../.." && pwd)"

"${ATTA_DIR}/scripts/infra-up.sh"
cd "${ROOT_DIR}"
exec turbo run dev dev:relay dev:events-consumer dev:jobs-consumer dev:scheduler \
  --filter=@atta/backend --filter=@atta/pwa
