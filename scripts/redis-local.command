#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${REDIS_PORT:-6379}"
ENV_FILE="${ROOT_DIR}/configs/.env"

env_value() {
  local key="$1"
  local fallback="$2"
  local current="${!key-}"
  if [[ -n "${current}" ]]; then
    echo "${current}"
    return 0
  fi
  if [[ ! -f "${ENV_FILE}" ]]; then
    echo "${fallback}"
    return 0
  fi
  local value
  value="$(awk -F= -v key="${key}" '
    $0 ~ /^[[:space:]]*#/ || $0 ~ /^[[:space:]]*$/ { next }
    {
      k=$1
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", k)
      if (k == key) {
        v=substr($0, index($0, "=") + 1)
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", v)
        print v
        exit
      }
    }
  ' "${ENV_FILE}")"
  echo "${value:-${fallback}}"
}

REDIS_ADDR="$(env_value REDIS_ADDR "")"
REDIS_PASSWORD="$(env_value REDIS_PASSWORD "")"

if [[ -n "${REDIS_ADDR:-}" && "${REDIS_ADDR}" == *:* ]]; then
  PORT="${REDIS_ADDR##*:}"
fi

clear

pids="$(lsof -tiTCP:"${PORT}" -sTCP:LISTEN || true)"
if [[ -n "${pids}" ]]; then
  echo "port ${PORT} is in use, stopping existing process(es): ${pids}"
  kill ${pids} || true
  sleep 1

  remaining="$(lsof -tiTCP:"${PORT}" -sTCP:LISTEN || true)"
  if [[ -n "${remaining}" ]]; then
    echo "process(es) still listening on ${PORT}, force stopping: ${remaining}"
    kill -9 ${remaining} || true
  fi
fi

echo "starting redis-server on port ${PORT}"
if [[ -n "${REDIS_PASSWORD:-}" ]]; then
  redis-server --port "${PORT}" --requirepass "${REDIS_PASSWORD}"
else
  redis-server --port "${PORT}"
fi