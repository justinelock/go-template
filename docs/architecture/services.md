# 服务职责

## gateway-service

**路径**：`cmd/gateway-service`、`internal/gateway/`

| 职责 | 说明 |
|------|------|
| 统一 HTTP 入口 | 对外 `:8180` |
| CORS | `withCORS` 中间件 |
| 鉴权 | token introspect → 注入 `X-User-Id` |
| 反向代理 | 按路由表转发到下游服务 |
| 健康检查 | `GET /healthz` |
| 公开配置 | `GET /v1/public/config` |

### 禁止

- 在 gateway transport 写业务规则（校验用户名、查库等）
- 绕过路由表硬编码下游 URL（应走 `resolve` + 服务名）
- 对浏览器暴露 member-service 端口替代网关

### 关键文件

- [`internal/gateway/transport/http/handler.go`](../../internal/gateway/transport/http/handler.go)
- [`internal/gateway/routes/routes.go`](../../internal/gateway/routes/routes.go) — 声明式路由表
- [`internal/gateway/app/auth.go`](../../internal/gateway/app/auth.go) — 鉴权与 token 提取

## member-service

**路径**：`cmd/member-service`、`internal/member/`

| 职责 | 说明 |
|------|------|
| 用户注册/登录/登出 | `app` + `repo` |
| Token 存储与 introspect | Redis |
| 用户资料 | GET/PUT profile |
| gRPC AuthService | 仅 `Introspect`（供网关） |

### 分层

```
transport/http|grpc  →  app  →  repo  →  domain
```

- transport：参数校验、错误码映射、`httpx.JSON`
- app：用例与 `domain.Err*`
- repo：MySQL / Redis

### 禁止

- 在 repo 返回 HTTP 状态码
- 手改 `api/gen/` 生成代码

## 新增服务（如 order-service）

复制 member-service 骨架，在网关 `routes.go` 增加代理条目。完整步骤见 [add-service.md](add-service.md)。

**order-service Demo**（v0.3）：`cmd/order-service`，演示 Redis 幂等/锁 + RabbitMQ 异步结算。详见 [order-demo.md](order-demo.md)。

| 职责 | 说明 |
|------|------|
| 创建订单 | POST `/v1/orders`，幂等键 + 分布式锁 |
| 查询订单 | GET `/v1/orders/{id}` |
| 异步结算 | 同进程 worker 经 `mq.Bus` 订阅 `order.settle`（RabbitMQ 默认 / RocketMQ 可切换） |
