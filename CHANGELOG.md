# Changelog

本文件记录本仓库所有 notable 变更。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号按需使用 [语义化版本](https://semver.org/lang/zh-CN/)。

**维护要求**：每次改代码、配置、文档或规则，在 `[Unreleased]` 下追加条目；发布时将 Unreleased 改为 dated 版本并清空 Unreleased 各节。

## [Unreleased]

### Added

- **GitHub Actions CI**：新增 [`.github/workflows/ci.yml`](.github/workflows/ci.yml)，包含 `test-and-build` 与 `compose-smoke` 两个 job。

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
