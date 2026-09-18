#!/usr/bin/env bash
# Wrap ng serve so SIGTERM/SIGINT shutdown exits 0 (Turbo/pnpm treat 130/143 as failure).
set -uo pipefail

child=""

shutdown() {
  if [[ -n "${child}" ]] && kill -0 "${child}" 2>/dev/null; then
    kill -INT "${child}" 2>/dev/null || true
    wait "${child}" 2>/dev/null || true
  fi
  exit 0
}

trap shutdown INT TERM

ng serve --host 0.0.0.0 --port 4200 &
child=$!

set +e
wait "${child}"
code=$?
set -e

case "${code}" in
0 | 130 | 143) exit 0 ;;
*) exit "${code}" ;;
esac
