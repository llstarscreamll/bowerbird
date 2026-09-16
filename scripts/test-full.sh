#!/usr/bin/env bash
set -euo pipefail

# Deterministic Go + PWA + e2e loop against a wiped local Postgres and .env.test.
# Usage: mise run test:full

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
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

ensure_env_test() {
  if [[ -f "${ROOT_DIR}/.env.test" ]]; then
    return 0
  fi
  if [[ ! -f "${ROOT_DIR}/.env.test.example" ]]; then
    die "missing .env.test and .env.test.example"
  fi
  log "Creating .env.test from .env.test.example"
  cp "${ROOT_DIR}/.env.test.example" "${ROOT_DIR}/.env.test"
}

reset_postgres() {
  log "Resetting Postgres volume"
  if docker inspect bowerbird-postgres >/dev/null 2>&1; then
    local vol
    vol="$(docker inspect bowerbird-postgres --format '{{range .Mounts}}{{if eq .Destination "/var/lib/postgresql/data"}}{{.Name}}{{end}}{{end}}')"
    docker compose stop postgres
    docker compose rm -f postgres
    if [[ -n "${vol}" ]]; then
      docker volume rm -f "${vol}"
    fi
  else
    docker volume rm -f bowerbird_postgres_data || true
  fi
}

ensure_env_test
export ENV_FILE="${ENV_FILE:-.env.test}"
if [[ "${ENV_FILE}" != /* ]]; then
  export ENV_FILE="${ROOT_DIR}/${ENV_FILE}"
fi
[[ -f "${ENV_FILE}" ]] || die "missing env file: ${ENV_FILE}"
log "Using ENV_FILE=${ENV_FILE}"

if curl -kf --max-time 2 "https://app.bowerbird.dev/api/health" >/dev/null 2>&1; then
  die "API already running on https://app.bowerbird.dev. Stop mise run dev, then retry."
fi

reset_postgres

log "Starting infra"
pnpm run infra:up

log "Migrating and seeding"
pnpm run migrate:all
pnpm run seed

log "Running Go and PWA tests"
turbo run test --filter=@bowerbird/backend --filter=@bowerbird/pwa

log "Starting API/PWA/workers for e2e"
turbo run dev dev:relay dev:events-consumer dev:jobs-consumer dev:scheduler \
  --filter=@bowerbird/backend --filter=@bowerbird/pwa &
APP_PID=$!
disown "${APP_PID}" 2>/dev/null || true

if ! wait_http "https://app.bowerbird.dev/api/health"; then
  die "timed out waiting for https://app.bowerbird.dev/api/health"
fi
if ! wait_http "https://app.bowerbird.dev/"; then
  die "timed out waiting for https://app.bowerbird.dev/"
fi

log "Installing Playwright browsers if needed"
pnpm run test:e2e:install

log "Running e2e tests (chromium + http; webkit skipped — use pnpm run test:e2e for full browser matrix)"
E2E_EXIT=0
pnpm run test:e2e:local || E2E_EXIT=$?

stop_app
trap - EXIT INT TERM

if [[ "${E2E_EXIT}" -ne 0 ]]; then
  die "e2e failed (exit ${E2E_EXIT})"
fi

log "Done"
exit 0
