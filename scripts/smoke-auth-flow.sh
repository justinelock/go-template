#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8180}"
NOW="$(date +%s)"
USERNAME="${USERNAME:-smoke_${NOW}}"
MOBILE="${MOBILE:-139${NOW: -8}}"
PASSWORD="${PASSWORD:-123456}"

json_request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local token="${4:-}"

  local args=(-sS -X "${method}" "${BASE_URL}${path}" -H "Content-Type: application/json")
  if [[ -n "${token}" ]]; then
    args+=(-H "Authorization: Bearer ${token}" -H "token: ${token}")
  fi
  if [[ -n "${body}" ]]; then
    args+=(-d "${body}")
  fi

  curl "${args[@]}"
}

json_field() {
  local json="$1"
  local path="$2"
  python3 - "$json" "$path" <<'PY'
import json
import sys

data = json.loads(sys.argv[1])
cur = data
for part in sys.argv[2].split("."):
    if not part:
        continue
    if isinstance(cur, dict):
        cur = cur.get(part, "")
    else:
        cur = ""
        break
print(cur if cur is not None else "")
PY
}

assert_code() {
  local name="$1"
  local response="$2"
  local expected="$3"
  local actual
  actual="$(json_field "${response}" "code")"
  if [[ "${actual}" != "${expected}" ]]; then
    echo "[fail] ${name}: expected code=${expected}, actual=${actual}" >&2
    echo "${response}" >&2
    exit 1
  fi
  echo "[pass] ${name}"
}

echo "[info] baseUrl=${BASE_URL}"
echo "[info] username=${USERNAME}"
echo "[info] mobile=${MOBILE}"

health="$(json_request GET /healthz)"
assert_code "health" "${health}" "0"

register_body="$(cat <<JSON
{
  "username": "${USERNAME}",
  "password": "${PASSWORD}",
  "mobile": "${MOBILE}",
  "email": "${USERNAME}@example.com",
  "nickname": "Smoke Test",
  "remark": "shell smoke test"
}
JSON
)"
register_resp="$(json_request POST /v1/auth/register "${register_body}")"
assert_code "register" "${register_resp}" "0"

login_body="$(cat <<JSON
{
  "username": "${USERNAME}",
  "password": "${PASSWORD}"
}
JSON
)"
login_resp="$(json_request POST /v1/auth/login "${login_body}")"
assert_code "login" "${login_resp}" "0"

token="$(json_field "${login_resp}" "data.accessToken")"
if [[ -z "${token}" ]]; then
  echo "[fail] login did not return data.accessToken" >&2
  echo "${login_resp}" >&2
  exit 1
fi

profile_resp="$(json_request GET /v1/member/users/profile "" "${token}")"
assert_code "profile" "${profile_resp}" "0"

logout_resp="$(json_request POST /v1/auth/logout "{}" "${token}")"
assert_code "logout" "${logout_resp}" "0"

after_logout_resp="$(json_request GET /v1/member/users/profile "" "${token}")"
assert_code "profile after logout" "${after_logout_resp}" "40102"

echo "[pass] auth flow completed"
