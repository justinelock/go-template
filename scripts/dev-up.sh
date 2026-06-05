#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

env_value() {
  local key="$1"
  local fallback="$2"
  local current="${!key-}"
  if [[ -n "${current}" ]]; then
    echo "${current}"
    return 0
  fi
  if [[ ! -f ./configs/.env ]]; then
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
  ' ./configs/.env)"
  echo "${value:-${fallback}}"
}

GATEWAY_SERVICE_PORT="$(env_value GATEWAY_SERVICE_PORT 8180)"
MEMBER_SERVICE_PORT="$(env_value MEMBER_SERVICE_PORT 8181)"
MEMBER_SERVICE_GRPC_PORT="$(env_value MEMBER_SERVICE_GRPC_PORT 9181)"
ORDER_SERVICE_PORT="$(env_value ORDER_SERVICE_PORT 8182)"

stop_port() {
  local port="$1"
  local pids
  pids="$(lsof -tiTCP:"${port}" -sTCP:LISTEN || true)"
  if [[ -z "${pids}" ]]; then
    return 0
  fi

  echo "port ${port} is in use, stopping process(es): ${pids}"
  kill ${pids} || true
  sleep 1

  pids="$(lsof -tiTCP:"${port}" -sTCP:LISTEN || true)"
  if [[ -n "${pids}" ]]; then
    echo "port ${port} still in use, force stopping process(es): ${pids}"
    kill -9 ${pids} || true
  fi
}

mkdir -p ./bin ./logs

# 步骤 1：编译 gateway、member、order 三个服务二进制。
echo "[build] compile gateway + member + order"
go build -o ./bin/gateway-service ./cmd/gateway-service
go build -o ./bin/member-service ./cmd/member-service
go build -o ./bin/order-service ./cmd/order-service

# 步骤 2：释放占用端口，避免 Listen 失败。
echo "[ports] ensure service ports are free"
stop_port "${GATEWAY_SERVICE_PORT}"
stop_port "${MEMBER_SERVICE_PORT}"
stop_port "${MEMBER_SERVICE_GRPC_PORT}"
stop_port "${ORDER_SERVICE_PORT}"

# 步骤 3：后台启动 gateway-service。
echo "[1/3] start gateway-service :${GATEWAY_SERVICE_PORT}"
./bin/gateway-service > ./logs/gateway-service.log 2>&1 &

# 步骤 4：后台启动 member-service（HTTP + gRPC）。
echo "[2/3] start member-service :${MEMBER_SERVICE_PORT} grpc :${MEMBER_SERVICE_GRPC_PORT}"
./bin/member-service > ./logs/member-service.log 2>&1 &

# 步骤 5：后台启动 order-service（需本机 RabbitMQ 可用）。
echo "[3/3] start order-service :${ORDER_SERVICE_PORT}"
./bin/order-service > ./logs/order-service.log 2>&1 &

echo ""
echo "gateway + member + order started."
echo "gateway: http://127.0.0.1:${GATEWAY_SERVICE_PORT}/healthz"
echo "logs:"
echo "  ./logs/member-service.log"
echo "  ./logs/gateway-service.log"
echo "  ./logs/order-service.log"
