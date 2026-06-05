#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8180}"
USER="pay$(date +%s)"
PASS="Pass123456"
MOBILE="13$(date +%s | tail -c 9)"

curl -fsS -X POST "${BASE_URL}/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${USER}\",\"password\":\"${PASS}\",\"mobile\":\"${MOBILE}\"}" >/dev/null

TOKEN=$(curl -fsS -X POST "${BASE_URL}/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${USER}\",\"password\":\"${PASS}\"}" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["accessToken"])')

IDEM="pay-idem-$(date +%s)"
ORDER_RESP=$(curl -fsS -X POST "${BASE_URL}/v1/order/orders" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "X-Idempotency-Key: ${IDEM}" \
  -H "Content-Type: application/json" \
  -d '{"product_id":"sku-pay-1","amount":19.9}')
ORDER_ID=$(echo "${ORDER_RESP}" | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["orderId"])')
echo "order created: ${ORDER_ID}"

sleep 2
for i in {1..25}; do
  STATUS=$(curl -fsS "${BASE_URL}/v1/order/orders/${ORDER_ID}" -H "Authorization: Bearer ${TOKEN}" \
    | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["status"])')
  echo "order status=${STATUS}"
  if [[ "${STATUS}" == "settled" ]]; then
    echo "payment flow ok"
    exit 0
  fi
  if [[ "${STATUS}" == "pending_payment" ]]; then
    PAY_ID=$(curl -fsS -X POST "${BASE_URL}/v1/payment/payments" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H "Content-Type: application/json" \
      -d "{\"order_id\":\"${ORDER_ID}\",\"amount\":19.9,\"channel\":\"mock\"}" \
      | python3 -c 'import sys,json; print(json.load(sys.stdin)["data"]["id"])')
    curl -fsS -X POST "${BASE_URL}/v1/payment/payments/${PAY_ID}/mock-pay" \
      -H "Authorization: Bearer ${TOKEN}" >/dev/null
    echo "mock paid payment ${PAY_ID}"
  fi
  sleep 1
done

echo "timeout waiting settled"
exit 1
