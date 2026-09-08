#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
USER="${ONPREM_HOST_USER:?}"
ADDRESS="${ONPREM_HOST_ADDRESS:?}"
PORT="${ONPREM_HOST_PORT:-22}"
REMOTE_DIR="${ONPREM_HOST_REMOTE_DIR:-/opt/bowerbird}"
REL="${ONPREM_RELEASE:?}"
APP_IMAGE="bowerbird-onprem-app:${REL}"
CADDY_IMAGE="bowerbird-onprem-caddy:${REL}"
REMOTE_COMPOSE="${REMOTE_DIR}/apps/deploy/onprem"
TARGET="${USER}@${ADDRESS}"

ssh_opts=(-o BatchMode=yes -o StrictHostKeyChecking=accept-new)
if [[ -n "${ONPREM_SSH_KEY_PATH:-}" ]]; then
  ssh_opts+=(-i "$ONPREM_SSH_KEY_PATH")
fi

ssh "${ssh_opts[@]}" -p "$PORT" "$TARGET" "mkdir -p '${REMOTE_COMPOSE}'"
scp "${ssh_opts[@]}" -P "$PORT" \
  "$ROOT/apps/deploy/onprem/docker-compose.yml" \
  "$ROOT/apps/deploy/onprem/Caddyfile" \
  "${TARGET}:${REMOTE_COMPOSE}/"

ssh "${ssh_opts[@]}" -p "$PORT" "$TARGET" \
  "test -f '${REMOTE_COMPOSE}/.env' || { echo 'missing ${REMOTE_COMPOSE}/.env on ${ADDRESS}' >&2; exit 1; }"

docker save "$APP_IMAGE" "$CADDY_IMAGE" | ssh "${ssh_opts[@]}" -p "$PORT" "$TARGET" docker load

ssh "${ssh_opts[@]}" -p "$PORT" "$TARGET" bash -s <<EOF
set -euo pipefail
cd '${REMOTE_COMPOSE}'
export ONPREM_RELEASE='${REL}'
docker compose run --rm --pull never migrate
docker compose up -d --no-build --remove-orphans --pull never
EOF
