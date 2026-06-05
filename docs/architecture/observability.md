# 可观测性（SMB 生产基线）

## 日志

- 使用标准库 `slog`，配置项 `LOG_LEVEL`、`LOG_FORMAT=json|text`
- 访问日志字段：`service`、`method`、`path`、`status`、`latency_ms`、`trace_id`、`user_id`（若有）
- 平台包：[`internal/platform/logging/`](../../internal/platform/logging/)

## 健康检查

| 路径 | 用途 |
|------|------|
| `GET /healthz` | Liveness |
| `GET /readyz` | Readiness（MySQL/Redis/MQ） |

## 指标

- `GET /metrics`：Prometheus RED + 业务 counter（`order_created_total`、`payment_paid_total` 等）
- Compose 可选 profile：`docker compose --profile obs up -d` 启动 Prometheus、Grafana、Jaeger

## 链路追踪

- 配置 `OTEL_ENABLED=true` 与 `OTEL_EXPORTER_OTLP_ENDPOINT`（如 `jaeger:4318`）
- 与现有 `X-Trace-Id` 并存；MQ 消息头 `traceId` 贯通 worker

## 本地验证

```bash
make compose-up
curl -s http://127.0.0.1:8180/readyz
curl -s http://127.0.0.1:8182/metrics | head
docker compose --profile obs up -d
open http://127.0.0.1:16686
```
