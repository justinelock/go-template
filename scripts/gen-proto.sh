#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT_DIR}/api/proto"
OUT_DIR="${ROOT_DIR}/api/gen"
MODULE_PATH="$(cd "${ROOT_DIR}" && go list -m -f '{{.Path}}')"

PROTOC_BIN="$(command -v protoc || true)"
PROTOC_GEN_GO_BIN="$(command -v protoc-gen-go || true)"
PROTOC_GEN_GO_GRPC_BIN="$(command -v protoc-gen-go-grpc || true)"

if [[ -z "${PROTOC_BIN}" || ! -x "${PROTOC_BIN}" ]]; then
  echo "error: protoc not found, please install Protocol Buffers compiler." >&2
  exit 1
fi

if [[ -z "${PROTOC_GEN_GO_BIN}" || ! -x "${PROTOC_GEN_GO_BIN}" ]]; then
  echo "error: protoc-gen-go not found, install with:" >&2
  echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" >&2
  exit 1
fi

if [[ -z "${PROTOC_GEN_GO_GRPC_BIN}" || ! -x "${PROTOC_GEN_GO_GRPC_BIN}" ]]; then
  echo "error: protoc-gen-go-grpc not found, install with:" >&2
  echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" >&2
  exit 1
fi

mkdir -p "${OUT_DIR}"

protoc \
  -I "${PROTO_DIR}" \
  --go_out="${OUT_DIR}" \
  --go_opt=paths=source_relative \
  --go_opt=Mmember/v1/auth.proto="${MODULE_PATH}/api/gen/member/v1" \
  --go-grpc_out="${OUT_DIR}" \
  --go-grpc_opt=paths=source_relative \
  --go-grpc_opt=Mmember/v1/auth.proto="${MODULE_PATH}/api/gen/member/v1" \
  "${PROTO_DIR}/member/v1/auth.proto"

echo "proto generation completed at ${OUT_DIR}"
