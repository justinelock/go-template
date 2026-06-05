# 项目文档索引

本目录按主题组织文档，便于扩展与 AI/新人检索。

| 目录 | 说明 | 状态 |
|------|------|------|
| [`architecture/`](architecture/README.md) | 服务拓扑、配置、本地开发、Redis 高并发、新增服务 playbook | 已维护 |
| [`api/`](api/README.md) | HTTP / gRPC / WebSocket API 规范 | 已维护 |
| [`conventions/`](conventions/code-comments.md) | 步骤级注释等开发约定 | 已维护 |

根目录 [`CHANGELOG.md`](../CHANGELOG.md)：每次代码/配置/文档变更须更新。

## 维护约定

- 修改 `internal/**/transport/http`、gRPC proto 或网关路由时，必须同步更新 [`api/error-codes.md`](api/error-codes.md) 及相关专题文档。
- 新增微服务时遵循 [`architecture/add-service.md`](architecture/add-service.md)。
- Cursor Agent 与开发者请先读 [`architecture/README.md`](architecture/README.md)、[`api/README.md`](api/README.md) 与根目录 [`AGENTS.md`](../AGENTS.md)。
