FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/gateway-service ./cmd/gateway-service \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/member-service ./cmd/member-service \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/order-service ./cmd/order-service \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/payment-service ./cmd/payment-service

FROM alpine:3.20
WORKDIR /srv
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /out /srv/bin
COPY configs /srv/configs

# default command can be overridden by docker compose
CMD ["/srv/bin/gateway-service"]
