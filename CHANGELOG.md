# Changelog

本文件记录本仓库所有 notable 变更。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号按需使用 [语义化版本](https://semver.org/lang/zh-CN/)。

**维护要求**：每次改代码、配置、文档或规则，在 `[Unreleased]` 下追加条目；发布时将 Unreleased 改为 dated 版本并清空 Unreleased 各节。

## [Unreleased]

### Fixed

- **api/gen/member/v1**：重新生成 `auth.pb.go`，修复损坏的 `rawDesc`（options 长度字节错误且缺 `role` 字段），消除 gateway/member 在 import proto 的 `init` 阶段 `slice bounds out of range [-4:]` 启动崩溃；补充 `TestFileDescriptorInit` 防止再漏检。

### Changed

- **configs/.env.example / configs/.env**：恢复并统一分组 `#` 注释；约定同步时不得剥离注释（见 `config-env.mdc`）。
- **步骤级注释补全**：为 v0.4–v0.5 新增 platform/payment/gateway/member/order 等包补全类型、字段与函数体 `// 步骤 N：` 注释。
- **order-service**：订单状态机 `pending_payment → paid → settled`；下单发 `payment.created`，支付后发 `order.settle`。
- **member-service**：密码 bcrypt + MD5 懒升级；refresh token 实装；`users.role` 简易 RBAC。
- **gateway-service**：限流、代理超时、断路器、RBAC（`X-User-Role`）、payment 路由。
- **GitHub Actions CI**：改为默认仅 `workflow_dispatch` 手动触发，不再 push/PR 自动运行；增强 compose 健康等待与 smoke 脚本权限。
- **Dockerfile**：构建镜像 `golang:1.25` → `1.26`，与 `go.mod` 一致（修复 compose 构建失败）。
- **步骤级注释**：为 order-service、platform（redislock/idempotency/mq）、网关 order 路由及 smoke 脚本补充步骤注释，对齐 `docs/conventions/code-comments.md`。
- **internal/platform/mq**：重构为统一 `Bus` 接口；`MQ_PROVIDER` 配置切换（默认 rabbitmq）；order-service 经 `order/mq.SettlePublisher` 适配，业务不依赖具体 MQ。

### Added

- **payment-service**：支付 Demo（MQ 建单、mock-pay、回调占位、`payment.paid`）。
- **platform 可观测**：`logging`、`health`（`/readyz`）、`metrics`、`otel`、`httpserver` 中间件链、`runtime` 优雅关闭。
- **platform 治理**：`ratelimit`、`gatewayresilience`（gobreaker）、`httpx` recovery。
- **docs/architecture/observability.md**、**security.md**、**payment-demo.md**、**production-baseline.md**。
- **api/openapi.yaml**：核心路径骨架。
- **scripts/smoke-payment-flow.sh**；Compose `payment-service` 与 `--profile obs`（Prometheus/Grafana/Jaeger）。
- **docs/architecture/mq.md**：MQ 使用指南（RabbitMQ/RocketMQ 接入、配置、手动建资源、order Demo 用法）。
- **internal/platform/mq/rocketmq.go**：RocketMQ `Bus` 实现。
- **internal/order/mq/publisher.go**：结算消息发布适配层。
- **docs/architecture/redis-high-concurrency.md**：Redis 高并发使用指南（Redisson 概念对照、分布式锁、幂等、多业务场景示例）。
- **order-service Demo**：Redis 幂等/分布式锁 + RabbitMQ 异步结算；平台包 `redislock`、`idempotency`、`mq`。
- **docs/architecture/order-demo.md**：订单 Demo 流程与 curl 示例。
- **scripts/smoke-order-flow.sh**：下单幂等与结算轮询冒烟。
- **docker-compose**：新增 `rabbitmq`、`order-service` 服务。
- **GitHub Actions CI**：新增 [`.github/workflows/ci.yml`](.github/workflows/ci.yml)，包含 `test-and-build` 与 `compose-smoke` 两个 job（默认手动触发）。

### Removed

- **docs/02-api/**：物理删除重复 API 文档目录，仅保留 `docs/api/` 作为唯一来源。

## [0.2.0] - 2026-06-04

### Added

- **docs/architecture/**：服务拓扑、配置说明、本地开发、**新增微服务 playbook**（`add-service.md`）。
- **docker-compose.yml**：MySQL、Redis、member-service、gateway-service 一键启动。
- **configs/app.dev.json**：本地 / Compose 环境配置。
- **Makefile**：`build`、`dev`、`compose-up/down`、`proto`、`test`、`smoke`。
- **internal/platform/errcode**：业务错误码与 message 常量。
- **internal/gateway/routes**：声明式代理路由表与 `ExtractToken`。
- **单元测试**：`httpx`、`config`、`routes`、`errcode`。
- **docs/api**、**docs/conventions**、**.cursor/rules**、**.cursor/skills**、**AGENTS.md**（v0.1 延续并扩展）。
- **internal/platform/ws**：WebSocket 占位；`GET /v1/ws` 返回 501 + `50101`。

### Changed

- **定位**：README 升级为「微服务框架模版」，含目录树与 v0.3 路线图。
- **gateway-service**：路由注册改为遍历 `routes.ProxyRoutes`；handler 使用 `errcode` 常量。
- **member-service**：HTTP handler 使用 `errcode` 常量（行为不变）。
- **模块名**：`pricing-assistant` → `go-template`。
- **配置**：`gateway_service_port` / `GATEWAY_SERVICE_PORT`（废弃 `API_GATEWAY_PORT`）。
- **Dockerfile**：镜像内包含 `configs/` 供 Compose 使用。

### Removed

- **docs/02-api/**：与 `docs/api/` 重复的文档目录。
- **configs/.env**：未使用的 `RABBITMQ_*`、`ORDER_SERVICE_URL`、`PUBLIC_WS_URL` 等变量。

### Docs

- **CHANGELOG** 与 **步骤级注释** 制度（`code-comments.mdc`）。

## [0.1.0] - 2026-06-04

### Added

- **gateway-service** + **member-service** 最小模板：注册、登录、登出、用户资料。
- 统一 JSON 信封（`code` / `message` / `data` / `trace_id`）。
- Redis 不透明 token（12h TTL）；网关 gRPC/HTTP introspect 鉴权。
- Postman 集合与 `scripts/smoke-auth-flow.sh` 冒烟脚本。

[Unreleased]: CHANGELOG.md#unreleased
[0.2.0]: CHANGELOG.md#020---2026-06-04
[0.1.0]: CHANGELOG.md#010---2026-06-04
