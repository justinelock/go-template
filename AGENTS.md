# Agent 指引

本仓库为 Go 后端模板（gateway-service + member-service）。自动化改代码时请遵循以下要点。

## 必读

1. 架构与加服务：[`docs/architecture/README.md`](docs/architecture/README.md)、[`docs/architecture/add-service.md`](docs/architecture/add-service.md)
2. API 规范索引：[`docs/api/README.md`](docs/api/README.md)
3. Cursor 规则：`.cursor/rules/`（HTTP / 错误码 / gRPC / **步骤注释** / **变更日志**）
4. 通用变更流程：`.cursor/skills/code-change/SKILL.md`；接口变更：`.cursor/skills/api-change/SKILL.md`
5. 变更日志：[`CHANGELOG.md`](CHANGELOG.md)（**每次任务结束必须更新** `[Unreleased]`）
6. 注释约定：[`docs/conventions/code-comments.md`](docs/conventions/code-comments.md)

## 结构

- `cmd/gateway-service`、`cmd/member-service`：入口
- `internal/gateway`：鉴权、CORS、反向代理（无业务）
- `internal/member`：用户与 token 用例
- `internal/platform/httpx`：统一 JSON 响应

## 代码风格

- 新增/修改函数：函数块注释 + 体内 **`// 步骤 1：`、`// 步骤 2：`…**（参考 `internal/member/transport/http/handler.go`）
- 结构体字段、错误分支、中间件链均需分步说明意图

## 禁止

- 跳过 gateway 对浏览器暴露 member HTTP 端口
- 完成改动却不更新 `CHANGELOG.md`
- 新增逻辑代码无任何步骤注释
- 在 gateway handler 写业务逻辑
- 新增错误码不更新 `docs/api/error-codes.md`
- 手改 `api/gen/` 生成代码

## 验证

```bash
make test
make smoke
```

Postman 集合：`api/postman_collection.json`
