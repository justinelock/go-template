# gRPC 约定

## 定位

- gRPC 用于 **服务间** 调用，**不对浏览器暴露**。
- 对外产品能力统一走 **HTTP gateway-service**。
- 当前仅实现 **`AuthService.Introspect`**，供网关校验 token。

## Proto 布局

| 文件 | 说明 |
|------|------|
| `api/proto/member/v1/auth.proto` | 服务定义 |
| `api/gen/member/v1/*.pb.go` | 生成代码，勿手改 |

约定：

- `package member.v1;`
- `go_package` 与模块路径一致（由 `scripts/gen-proto.sh` 注入 `-M` 映射）

## AuthService.Introspect

请求 `IntrospectRequest`：

| 字段 | 类型 | 说明 |
|------|------|------|
| `token` | string | 访问令牌 |
| `trace_id` | string | 与 HTTP `X-Trace-Id` 对齐 |

响应 `IntrospectResponse`（成功）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int32 | `0` 成功 |
| `message` | string | `ok` |
| `user_id` | string | 用户 ID |
| `trace_id` | string | 回显 |
| `role` | string | 用户角色（简易 RBAC） |

### gRPC status vs 消息内 code

| 情况 | gRPC code | 消息体 |
|------|-----------|--------|
| token 为空 | `InvalidArgument` | 无成功体 |
| token 无效 | `Unauthenticated` | 无成功体 |
| 内部错误 | `Internal` | 无成功体 |
| 成功 | OK | `code=0`, `user_id` 填充 |

实现：`internal/member/transport/grpc/auth_server.go`。

网关客户端：`internal/gateway/client/membergrpc/client.go`，超时 **800ms**，失败回退 HTTP `GET /v1/auth/introspect`。

## 何时新增 RPC

适合：

- 高频、低延迟的内部调用（鉴权、批量查询、事件推送侧车）

不适合：

- 直接替代对外 REST（应仍经 gateway 暴露 HTTP）

新增步骤：

1. 在 `api/proto/<domain>/v1/` 增加 `.proto` 并定义 service。
2. 运行 `./scripts/gen-proto.sh`。
3. 在对应服务 `internal/<domain>/transport/grpc/` 实现 server。
4. 更新本文档与 [error-codes.md](error-codes.md)（若响应含业务码）。
5. 在 gateway 或调用方增加 client 与超时/回退策略。

## 生成命令

依赖：`protoc`、`protoc-gen-go`、`protoc-gen-go-grpc`。

```bash
./scripts/gen-proto.sh
```

生成输出目录：`api/gen/`。

## 配置

- `member-service` 监听 gRPC 端口（默认 **9181**）。
- Gateway 是否优先 gRPC 由配置项 `gateway_use_member_grpc` 控制（见 `configs/app.*.json`）。
