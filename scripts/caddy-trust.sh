#!/usr/bin/env bash

set -euo pipefail

readonly CADDY_CONTAINER_NAME="bowerbird-caddy"
readonly CADDY_CA_PATH_IN_CONTAINER="/data/caddy/pki/authorities/local/root.crt"
readonly LINUX_SYSTEM_CA_TARGET_FEDORA="/etc/pki/ca-trust/source/anchors/bowerbird-caddy-local-ca.crt"
readonly LINUX_SYSTEM_CA_TARGET_ARCH="/etc/ca-certificates/trust-source/anchors/bowerbird-caddy-local-ca.crt"
readonly NICKNAME="Bowerbird Caddy Local CA"

TMP_DIR=""

log() {
  printf '[caddy-trust] %s\n' "$*"
}

warn() {
  printf '[caddy-trust] WARN: %s\n' "$*" >&2
}

die() {
  printf '[caddy-trust] ERROR: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "${TMP_DIR:-}" && -d "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
  fi
}

require_command() {
  local cmd="$1"
  command -v "$cmd" >/dev/null 2>&1 || die "Missing required command: $cmd"
}

extract_caddy_root_ca() {
  local output_file="$1"

  require_command docker

  if ! docker inspect "$CADDY_CONTAINER_NAME" >/dev/null 2>&1; then
    die "Container '$CADDY_CONTAINER_NAME' was not found. Start infra first (for example: pnpm run infra:up)."
  fi

  local running
  running="$(docker inspect -f '{{.State.Running}}' "$CADDY_CONTAINER_NAME" 2>/dev/null || true)"
  if [[ "$running" != "true" ]]; then
    die "Container '$CADDY_CONTAINER_NAME' is not running. Start infra first (for example: pnpm run infra:up)."
  fi

  if ! docker cp "$CADDY_CONTAINER_NAME:$CADDY_CA_PATH_IN_CONTAINER" "$output_file" >/dev/null 2>&1; then
    die "Could not copy Caddy root CA from '$CADDY_CA_PATH_IN_CONTAINER'."
  fi
}

upsert_nss_certificate() {
  local db_dir="$1"
  local cert_file="$2"

  [[ -d "$db_dir" ]] || mkdir -p "$db_dir"

  if certutil -L -d "sql:$db_dir" -n "$NICKNAME" >/dev/null 2>&1; then
    certutil -D -d "sql:$db_dir" -n "$NICKNAME" >/dev/null 2>&1 || true
  fi

  certutil -A -d "sql:$db_dir" -n "$NICKNAME" -t "C,," -i "$cert_file"
}

import_firefox_profiles() {
  local profiles_root="$1"
  local cert_file="$2"
  local imported_any="false"

  [[ -d "$profiles_root" ]] || return 0

  for profile_dir in "$profiles_root"/*; do
    [[ -d "$profile_dir" ]] || continue

    if [[ -f "$profile_dir/cert9.db" ]]; then
      upsert_nss_certificate "$profile_dir" "$cert_file"
      imported_any="true"
      log "Imported certificate into Firefox profile: $profile_dir"
    fi
  done

  if [[ "$imported_any" == "false" ]]; then
    warn "No Firefox profiles with cert9.db found at: $profiles_root"
  fi
}

install_linux_fedora() {
  local cert_file="$1"

  require_command sudo
  require_command update-ca-trust

  log "Installing Caddy root CA into Fedora system trust store"
  sudo install -D -m 0644 "$cert_file" "$LINUX_SYSTEM_CA_TARGET_FEDORA"
  sudo update-ca-trust

  install_nss_databases "$cert_file"
}

install_linux_arch() {
  local cert_file="$1"

  require_command sudo
  require_command trust

  log "Installing Caddy root CA into Arch Linux system trust store"
  sudo install -D -m 0644 "$cert_file" "$LINUX_SYSTEM_CA_TARGET_ARCH"
  sudo trust extract-compat

  install_nss_databases "$cert_file"
}

install_nss_databases() {
  local cert_file="$1"

  if command -v certutil >/dev/null 2>&1; then
    log "Installing certificate into NSS databases (Chrome/Chromium/Firefox)"
    upsert_nss_certificate "$HOME/.pki/nssdb" "$cert_file"

    if [[ -d "$HOME/.var/app/org.chromium.Chromium" ]]; then
      upsert_nss_certificate "$HOME/.var/app/org.chromium.Chromium/.pki/nssdb" "$cert_file"
    fi

    if [[ -d "$HOME/.var/app/com.google.Chrome" ]]; then
      upsert_nss_certificate "$HOME/.var/app/com.google.Chrome/.pki/nssdb" "$cert_file"
    fi

    import_firefox_profiles "$HOME/.mozilla/firefox" "$cert_file"
    import_firefox_profiles "$HOME/.var/app/org.mozilla.firefox/.mozilla/firefox" "$cert_file"
  else
    warn "certutil is not installed; browser-specific trust steps were skipped."
    warn "Install it with your package manager (e.g., sudo dnf install nss-tools or sudo pacman -S nss)"
  fi
}

install_macos() {
  local cert_file="$1"

  require_command sudo
  require_command security

  log "Installing Caddy root CA into macOS System keychain"
  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$cert_file"

  if command -v certutil >/dev/null 2>&1; then
    log "Installing certificate into Firefox profiles"
    import_firefox_profiles "$HOME/Library/Application Support/Firefox/Profiles" "$cert_file"
  else
    warn "certutil is not installed; Firefox profile import was skipped."
    warn "Install NSS tools if needed (for example: brew install nss)."
  fi
}

main() {
  local os
  os="$(uname -s)"

  TMP_DIR="$(mktemp -d)"
  trap cleanup EXIT

  local cert_file="$TMP_DIR/caddy-root.crt"
  log "Extracting Caddy local root certificate from Docker"
  extract_caddy_root_ca "$cert_file"

  case "$os" in
    Linux)
      if [[ -f /etc/os-release ]]; then
        source /etc/os-release
        if [[ "${ID:-}" == "arch" || "${ID_LIKE:-}" == *"arch"* ]]; then
          install_linux_arch "$cert_file"
        elif [[ "${ID:-}" == "fedora" || "${ID_LIKE:-}" == *"fedora"* || "${ID_LIKE:-}" == *"rhel"* ]]; then
          install_linux_fedora "$cert_file"
        else
          die "Unsupported Linux distribution. Please manually trust the certificate."
        fi
      else
        die "Cannot detect Linux distribution (/etc/os-release missing)."
      fi
      ;;
    Darwin)
      install_macos "$cert_file"
      ;;
    *)
      die "Unsupported OS: $os"
      ;;
  esac

  log "Done. Restart browsers that were open before running this script."
}

main "$@"
