#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

log() {
  printf '[setup-local] %s\n' "$*"
}

die() {
  printf '[setup-local] ERROR: %s\n' "$*" >&2
  exit 1
}

require_command() {
  local cmd="$1"
  command -v "$cmd" >/dev/null 2>&1 || die "Missing required command: $cmd"
}

run_with_mise() {
  mise exec -- "$@"
}

install_mise_tools() {
  log "Installing mise toolchain from .mise.toml"
  mise install
}

install_workspace_deps() {
  log "Installing workspace dependencies (pnpm install)"
  run_with_mise pnpm install
}

install_agent_skills() {
  log "Installing agent skills"

  log "  GoogleChrome/modern-web-guidance"
  run_with_mise npx --yes skills add GoogleChrome/modern-web-guidance --all

  log "  spartan-ng/spartan"
  run_with_mise npx --yes skills add spartan-ng/spartan --all

  log "  tech-leads-club agent-skills"
  run_with_mise npx --yes @tech-leads-club/agent-skills install \
    -s docs-writer coding-guidelines tactical-ddd modular-design-principles tlc-spec-driven aws-advisor

  log "  OpenSpec CLI + project instructions"
  run_with_mise npm install -g @fission-ai/openspec@latest
  run_with_mise openspec init --tools cursor,opencode,claude,agents --no-animation --force
}

install_mcp_servers() {
  log "Installing MCP servers"

  log "  @spartan-ng/mcp (spartan-mcp)"
  run_with_mise npm install -g @spartan-ng/mcp

  if [[ ! -f "$ROOT_DIR/.cursor/mcp.json" ]]; then
    die "Missing project Cursor MCP config at .cursor/mcp.json"
  fi
  if [[ ! -f "$ROOT_DIR/opencode.json" ]]; then
    die "Missing project OpenCode MCP config at opencode.json"
  fi

  log "Cursor/OpenCode project MCP configs already present"
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'EOF'
Usage: setup-local.sh

Installs the local development toolchain and agent tooling for this repo:
  1. mise install (.mise.toml)
  2. pnpm install
  3. Agent skills (Chrome guidance, Spartan, TLC, OpenSpec)
  4. MCP CLIs (@spartan-ng/mcp) and verifies project MCP configs

Requires: mise on PATH

Run via: pnpm run setup:local
EOF
    exit 0
  fi

  require_command mise

  log "Repo root: $ROOT_DIR"

  install_mise_tools
  install_workspace_deps
  install_agent_skills
  install_mcp_servers

  log "Done. Next: cp .env.example .env (repo root), then pnpm run infra:up / pnpm run dev"
  log "See docs/technical/getting-started.md"
}

main "$@"
