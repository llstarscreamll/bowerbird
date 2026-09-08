#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$ROOT/apps/deploy/onprem"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  . "$ROOT/.env"
  set +a
fi

HOSTS_FILE="${ONPREM_HOSTS_FILE:-$ROOT/apps/deploy/onprem/hosts.json}"

host_count() {
  HOSTS_FILE="$HOSTS_FILE" node --input-type=module -e '
import fs from "node:fs";
const p = process.env.HOSTS_FILE ?? "";
if (!fs.existsSync(p)) process.exit(2);
const raw = JSON.parse(fs.readFileSync(p, "utf8"));
const list = Array.isArray(raw) ? raw : raw.hosts;
process.exit(Array.isArray(list) && list.length > 0 ? 0 : 1);
'
}

if ! host_count; then
  echo "on-prem fleet: no hosts in ${HOSTS_FILE}, skip"
  exit 0
fi

bash "$ROOT/apps/deploy/onprem/scripts/build-images.sh"
pulumi stack select --create fleet
pulumi up --yes
