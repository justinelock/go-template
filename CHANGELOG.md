# Changelog

本文件记录本仓库所有 notable 变更。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号按需使用 [语义化版本](https://semver.org/lang/zh-CN/)。

**维护要求**：每次改代码、配置、文档或规则，在 `[Unreleased]` 下追加条目；发布时将 Unreleased 改为 dated 版本并清空 Unreleased 各节。

## [Unreleased]

### Added

- **docs/api**：API 规范（REST、认证、错误码、gRPC、WebSocket）。
- **docs/conventions**：步骤级注释约定文档。
- **.cursor/rules**：`code-comments.mdc`、`changelog.mdc`（及既有 api-http / api-errors / api-grpc）。
- **.cursor/skills**：`api-change` 接口变更检查清单；`code-change` 通用变更流程。
- **AGENTS.md**：Agent 必读指引。
- **internal/platform/ws**：WebSocket `Hub` 接口与 `NoopHub` 占位。
- **gateway**：`GET /v1/ws` 占位，HTTP 501 + 业务码 `50101`。

### Changed

- **gateway-service**：服务由 `api-gateway` 更名为 `gateway-service`（`cmd/gateway-service`、health、脚本、文档、Docker）；配置仅使用 `gateway_service_port` / `GATEWAY_SERVICE_PORT`（废弃 `api_gateway_port` / `API_GATEWAY_PORT`）。
- **模块名**：`pricing-assistant` → `go-template`（`go.mod`、import 路径、proto `go_package`）。
- **文档目录**：API 文档统一为 `docs/api/`（非服务名 `02-api`）。
- **README**：Postman 路径修正为 `api/postman_collection.json`，并链接 API 文档。

### Docs

- **CHANGELOG.md**：建立变更日志制度。
- **docs/conventions/code-comments.md**：步骤级注释细则与示例。

### Removed

- **配置**：移除 `API_GATEWAY_PORT`、`api_gateway_port` 兼容逻辑。

### Chore

- **configs/.env**：与 `.env.example` 结构同步（保留本地 prod 取值）。
- **.cursor/rules**：新增 `code-comments.mdc`、`changelog.mdc`（`alwaysApply: true`）。
- **AGENTS.md** / **api-change** Skill：强制步骤注释与 CHANGELOG 更新。

## [0.1.0] - 2026-06-04

### Added

- **gateway-service** + **member-service** 最小模板：注册、登录、登出、用户资料。
- 统一 JSON 信封（`code` / `message` / `data` / `trace_id`）。
- Redis 不透明 token（12h TTL）；网关 gRPC/HTTP introspect 鉴权。
- Postman 集合与 `scripts/smoke-auth-flow.sh` 冒烟脚本。

[Unreleased]: https://github.com/your-org/go-template/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/your-org/go-template/releases/tag/v0.1.0
