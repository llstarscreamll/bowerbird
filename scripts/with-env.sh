#!/usr/bin/env bash
# Load ENV_FILE (default .env at repo root). When executed, run the remaining args.
# Usage: with-env.sh <command> [args...]
#        source scripts/with-env.sh

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
fi

_with_env_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_repo_root="$(cd "${_with_env_dir}/.." && pwd)"
_env_file="${ENV_FILE:-.env}"
if [[ "${_env_file}" != /* ]]; then
  _env_file="${_repo_root}/${_env_file}"
fi

if [[ ! -f "${_env_file}" ]]; then
  printf 'with-env: missing env file: %s (ENV_FILE=%s)\n' "${_env_file}" "${ENV_FILE:-.env}" >&2
  unset _with_env_dir _repo_root _env_file
  if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    return 1
  fi
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "${_env_file}"
set +a
export ENV_FILE="${_env_file}"
unset _with_env_dir _repo_root _env_file

if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
  return 0
fi

if [[ $# -eq 0 ]]; then
  printf 'with-env: usage: with-env.sh <command> [args...]\n' >&2
  exit 2
fi

exec "$@"
