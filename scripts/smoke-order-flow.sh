#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8180}"
NOW="$(date +%s)"
USERNAME="${USERNAME:-order_smoke_${NOW}}"
MOBILE="${MOBILE:-138${NOW: -8}}"
PASSWORD="${PASSWORD:-123456}"
IDEMPOTENCY_KEY="${IDEMPOTENCY_KEY:-order-demo-${NOW}}"

json_request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local token="${4:-}"
  local idempotency="${5:-}"

  local args=(-sS -X "${method}" "${BASE_URL}${path}" -H "Content-Type: application/json")
  if [[ -n "${token}" ]]; then
    args+=(-H "Authorization: Bearer ${token}" -H "token: ${token}")
  fi
  if [[ -n "${idempotency}" ]]; then
    args+=(-H "X-Idempotency-Key: ${idempotency}")
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

echo "[info] order smoke baseUrl=${BASE_URL}"

# 步骤 1：注册测试用户。
register_body="$(cat <<JSON
{
  "username": "${USERNAME}",
  "password": "${PASSWORD}",
  "mobile": "${MOBILE}",
  "email": "${USERNAME}@example.com",
  "nickname": "Order Smoke"
}
JSON
)"
register_resp="$(json_request POST /v1/auth/register "${register_body}")"
assert_code "register for order" "${register_resp}" "0"

# 步骤 2：登录并提取 accessToken。
login_body="$(cat <<JSON
{
  "username": "${USERNAME}",
  "password": "${PASSWORD}"
}
JSON
)"
login_resp="$(json_request POST /v1/auth/login "${login_body}")"
assert_code "login for order" "${login_resp}" "0"

token="$(json_field "${login_resp}" "data.accessToken")"
if [[ -z "${token}" ]]; then
  echo "[fail] login did not return token" >&2
  exit 1
fi

# 步骤 3：创建订单（带幂等键），期望 status=pending。
order_body='{"product_id":"sku-1","amount":99.00}'
create_resp="$(json_request POST /v1/order/orders "${order_body}" "${token}" "${IDEMPOTENCY_KEY}")"
assert_code "create order" "${create_resp}" "0"

order_id="$(json_field "${create_resp}" "data.orderId")"
status="$(json_field "${create_resp}" "data.status")"
if [[ -z "${order_id}" ]]; then
  echo "[fail] create order missing orderId" >&2
  echo "${create_resp}" >&2
  exit 1
fi
if [[ "${status}" != "pending" ]]; then
  echo "[fail] expected pending status, got ${status}" >&2
  exit 1
fi
echo "[pass] create order orderId=${order_id} status=${status}"

# 步骤 4：相同幂等键重复下单，应返回同一 orderId。
dup_resp="$(json_request POST /v1/order/orders "${order_body}" "${token}" "${IDEMPOTENCY_KEY}")"
assert_code "idempotent create order" "${dup_resp}" "0"
dup_order_id="$(json_field "${dup_resp}" "data.orderId")"
if [[ "${dup_order_id}" != "${order_id}" ]]; then
  echo "[fail] idempotency: expected ${order_id}, got ${dup_order_id}" >&2
  exit 1
fi
echo "[pass] idempotency returns same orderId"

# 步骤 5：轮询订单详情，直到 MQ worker 将 status 更新为 settled。
settled=""
for i in {1..30}; do
  get_resp="$(json_request GET "/v1/order/orders/${order_id}" "" "${token}")"
  current_status="$(json_field "${get_resp}" "data.status")"
  if [[ "${current_status}" == "settled" ]]; then
    settled="yes"
    break
  fi
  sleep 1
done

if [[ -z "${settled}" ]]; then
  echo "[fail] order not settled within timeout" >&2
  exit 1
fi
echo "[pass] order settled orderId=${order_id}"

echo "[pass] order flow completed"
