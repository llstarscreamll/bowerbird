#!/usr/bin/env bash
set -euo pipefail

# Deterministic Atta Go + web + e2e loop against a wiped local Postgres
# and apps/atta/.env.test. Usage: mise //apps/atta:test:full

ATTA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_DIR="$(cd "${ATTA_DIR}/../.." && pwd)"
cd "${ROOT_DIR}"

log() {
  printf '[test:full] %s\n' "$*"
}

die() {
  printf '[test:full] ERROR: %s\n' "$*" >&2
  exit 1
}

kill_tree() {
  local pid="${1:-}"
  [[ -n "${pid}" ]] || return 0
  local child
  while read -r child; do
    [[ -n "${child}" ]] || continue
    kill_tree "${child}"
  done < <(ps -o pid= --ppid "${pid}" 2>/dev/null || true)
  kill "${pid}" 2>/dev/null || true
}

wait_http() {
  local url="$1"
  local started="${SECONDS}"
  while ((SECONDS - started < 180)); do
    if curl -kf --max-time 2 "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  return 1
}

APP_PID=""

stop_app() {
  if [[ -z "${APP_PID}" ]]; then
    return 0
  fi
  log "Stopping API/PWA/workers (pid ${APP_PID})"
  # SIGINT lets Turbo forward graceful shutdown (like Ctrl+C); SIGTERM leaf-first kills ng with 143.
  kill -INT "${APP_PID}" 2>/dev/null || true
  local i
  for ((i = 0; i < 15; i++)); do
    kill -0 "${APP_PID}" 2>/dev/null || break
    sleep 1
  done
  if kill -0 "${APP_PID}" 2>/dev/null; then
    kill_tree "${APP_PID}"
  fi
  wait "${APP_PID}" 2>/dev/null || true
  APP_PID=""
}

cleanup() {
  stop_app
}
trap cleanup EXIT INT TERM

ATTA_COMPOSE=(docker compose -f "${ATTA_DIR}/docker-compose.yml")

ensure_env_test() {
  if [[ -f "${ATTA_DIR}/.env.test" ]]; then
    return 0
  fi
  if [[ ! -f "${ATTA_DIR}/.env.test.example" ]]; then
    die "missing ${ATTA_DIR}/.env.test and .env.test.example"
  fi
  log "Creating apps/atta/.env.test from .env.test.example"
  cp "${ATTA_DIR}/.env.test.example" "${ATTA_DIR}/.env.test"
}

reset_postgres() {
  log "Resetting Postgres volume"
  docker rm -f bowerbird-postgres bowerbird-minio bowerbird-rabbitmq bowerbird-caddy \
    canopy-postgres canopy-minio canopy-rabbitmq canopy-caddy \
    atta-postgres atta-minio atta-rabbitmq atta-caddy >/dev/null 2>&1 || true
  docker volume rm -f bowerbird_postgres_data bowerbird_minio_data bowerbird_caddy_data bowerbird_caddy_config \
    canopy_postgres_data canopy_minio_data canopy_caddy_data canopy_caddy_config \
    atta_postgres_data atta_minio_data atta_caddy_data atta_caddy_config >/dev/null 2>&1 || true
  if docker inspect atta-postgres >/dev/null 2>&1; then
    local vol
    vol="$(docker inspect atta-postgres --format '{{range .Mounts}}{{if eq .Destination "/var/lib/postgresql/data"}}{{.Name}}{{end}}{{end}}')"
    "${ATTA_COMPOSE[@]}" stop postgres
    "${ATTA_COMPOSE[@]}" rm -f postgres
    if [[ -n "${vol}" ]]; then
      docker volume rm -f "${vol}"
    fi
  else
    docker volume rm -f atta_postgres_data || true
  fi
}

ensure_env_test
export ENV_FILE="${ENV_FILE:-apps/atta/.env.test}"
if [[ "${ENV_FILE}" != /* ]]; then
  export ENV_FILE="${ROOT_DIR}/${ENV_FILE}"
fi
[[ -f "${ENV_FILE}" ]] || die "missing env file: ${ENV_FILE}"
log "Using ENV_FILE=${ENV_FILE}"

if curl -kf --max-time 2 "https://app.atta.dev/api/health" >/dev/null 2>&1; then
  die "API already running on https://app.atta.dev. Stop mise //apps/atta:dev, then retry."
fi

reset_postgres

export_caddy_ca() {
  local cert="${ROOT_DIR}/.cache/atta-caddy-root.crt"
  local bundle="${ROOT_DIR}/.cache/atta-ca-bundle.pem"
  mkdir -p "${ROOT_DIR}/.cache"
  docker cp atta-caddy:/data/caddy/pki/authorities/local/root.crt "${cert}" >/dev/null
  local sys_bundle=""
  if [[ -f /etc/ssl/certs/ca-certificates.crt ]]; then
    sys_bundle=/etc/ssl/certs/ca-certificates.crt
  elif [[ -f /etc/ssl/cert.pem ]]; then
    sys_bundle=/etc/ssl/cert.pem
  fi
  if [[ -n "${sys_bundle}" ]]; then
    cat "${sys_bundle}" "${cert}" > "${bundle}"
    export SSL_CERT_FILE="${bundle}"
  else
    export SSL_CERT_FILE="${cert}"
  fi
  export NODE_EXTRA_CA_CERTS="${cert}"
  log "Using Caddy local CA via SSL_CERT_FILE=${SSL_CERT_FILE}"
}

log "Starting infra"
"${ATTA_DIR}/scripts/infra-up.sh"
export_caddy_ca

log "Migrating and seeding"
pnpm --filter @atta/backend run migrate:all
pnpm --filter @atta/backend run seed

log "Running Go and PWA tests"
turbo run test --filter=@atta/backend --filter=@atta/pwa

log "Starting API/PWA/workers for e2e"
turbo run dev dev:relay dev:events-consumer dev:jobs-consumer dev:scheduler \
  --filter=@atta/backend --filter=@atta/pwa &
APP_PID=$!
disown "${APP_PID}" 2>/dev/null || true

if ! wait_http "https://app.atta.dev/api/health"; then
  die "timed out waiting for https://app.atta.dev/api/health"
fi
if ! wait_http "https://app.atta.dev/"; then
  die "timed out waiting for https://app.atta.dev/"
fi

log "Installing Playwright browsers if needed"
pnpm --filter @atta/e2e run test:e2e:install

log "Running e2e tests (chromium + http; webkit skipped — use mise //apps/atta:test:e2e for full browser matrix)"
E2E_EXIT=0
pnpm --filter @atta/e2e run test:e2e:local || E2E_EXIT=$?

stop_app
trap - EXIT INT TERM

if [[ "${E2E_EXIT}" -ne 0 ]]; then
  die "e2e failed (exit ${E2E_EXIT})"
fi

log "Done"
exit 0
