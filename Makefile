SHELL := /bin/bash

.PHONY: build dev compose-up compose-down proto test smoke tidy

build:
	@mkdir -p bin logs
	go build -o ./bin/gateway-service ./cmd/gateway-service
	go build -o ./bin/member-service ./cmd/member-service
	go build -o ./bin/order-service ./cmd/order-service

dev:
	@bash scripts/dev-up.sh

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

proto:
	@bash scripts/gen-proto.sh

test:
	go test ./...

smoke:
	@bash scripts/smoke-auth-flow.sh
	@bash scripts/smoke-order-flow.sh

tidy:
	go mod tidy
