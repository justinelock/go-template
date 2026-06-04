---
name: api-change
description: >-
  新增或修改 HTTP/gRPC 接口时的检查清单。在改 handler、proto、网关路由或错误码时使用。
---

# API 变更流程

按顺序执行，完成后跑冒烟测试。

## 1. 读规范

- [docs/api/README.md](../../../docs/api/README.md)
- 相关专题：REST / auth / error-codes / grpc / websocket

## 2. 设计

- 对外路径是否经 **gateway** `/v1/...`？
- 是否需要网关鉴权？更新 `internal/gateway/app/auth.go` 的 `RequiresAuth`。
- 网关路径与 member 内部路径映射是否在 `gateway/transport/http/handler.go` 注册？

## 3. 实现

| 层级 | 位置 |
|------|------|
| 路由 | `internal/{svc}/transport/http/handler.go` 或 gateway 代理 |
| 用例 | `internal/member/app/` |
| 持久化 | `internal/member/repo/` |
| 响应 | `httpx.JSON` + `internal/platform/httpx` |

gRPC：改 `api/proto/` → `./scripts/gen-proto.sh` → `transport/grpc/`。

## 4. 错误码

1. 在 [docs/api/error-codes.md](../../../docs/api/error-codes.md) **先登记**新码。
2. 在 handler 使用；禁止复用已有码。
3. app 层继续返回 `domain.Err*`，transport 映射。

## 5. 文档与集合

- 更新 `docs/api/README.md` 接口表（如有新端点）
- 更新 `api/postman_collection.json`（可选但推荐）

## 6. 步骤级注释

- 所有新增/修改的 handler、app、repo、gateway 逻辑须按 `docs/conventions/code-comments.md` 写 **`// 步骤 N：`**。
- 导出函数写块注释；错误映射、鉴权、代理等每一步单独注释。

## 7. 变更日志

- 在根目录 [`CHANGELOG.md`](../../../CHANGELOG.md) 的 **`## [Unreleased]`** 下追加条目（Added / Changed / Fixed / Docs 等）。
- 一条变更一行，注明范围（gateway-service / member / docs/api …）与行为说明。

## 8. 验证

```bash
go build ./...
./scripts/smoke-auth-flow.sh
```

服务未启动时先：`./scripts/dev-up.sh`

## 禁止

- gateway 写业务逻辑
- 手改 `api/gen/`
- 未登记错误码就合并
- 未更新 CHANGELOG.md
- 新增代码无步骤级注释
