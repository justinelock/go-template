# 错误码规范

## 编码规则

业务码为 **整数**，与 HTTP 状态码独立。建议按段分配：

| 段 | 含义 | 示例 |
|----|------|------|
| `0` | 成功 | `0` + `message: ok` |
| `405` | HTTP 方法不允许 | 与 HTTP 405 同时使用 |
| `40001`–`40099` | 客户端 / 参数 / 冲突 | 末两位区分具体场景 |
| `40101`–`40199` | 认证与授权 | 缺 token、无效 token、凭证错误 |
| `40401`–`40499` | 资源不存在 | |
| `50001`–`50099` | 服务端内部错误 | 含网关代理失败 |
| `50101`–`50199` | 未实现 / 规划中能力 | WebSocket 占位等 |

### 新增错误码流程

1. 在本文档登记：码值、HTTP status、`message`、触发场景、服务。
2. 在 handler 中使用，**禁止复用**已占用码值。
3. 若多服务共用码段，在表中注明 `gateway-service` / `member`。

后续可抽到 `internal/platform/errcode` 集中常量（单独 PR）。

## 完整对照表（当前实现）

### 通用

| code | HTTP | message | 服务 | 场景 |
|------|------|---------|------|------|
| 0 | 200 | ok | gateway-service / member | 成功 |

### 4xx 客户端

| code | HTTP | message | 服务 | 场景 |
|------|------|---------|------|------|
| 405 | 405 | method not allowed | member | 非允许 HTTP 方法 |
| 40001 | 400 | invalid request body | member | 登录 JSON 解析失败 |
| 40002 | 400 | username/password is required | member | 登录缺少账号或密码 |
| 40011 | 400 | invalid request body | member | 注册 JSON 解析失败 |
| 40012 | 400 | username/password/mobile is required | member | 注册必填缺失 |
| 40013 | 400 | username already exists | member | 注册用户名冲突 |
| 40014 | 400 | invalid request body | member | 资料更新 JSON 解析失败 |
| 40015 | 400 | at least one field is required | member | 资料更新无有效字段 |
| 40016 | 400 | mobile already exists | member | 注册手机号冲突 |
| 40101 | 401 | token is required | gateway-service / member | 缺 token |
| 40102 | 401 | token is invalid or expired | gateway-service / member | token 无效或过期 |
| 40103 | 401 | username or password is invalid | member | 登录凭证错误 |
| 40402 | 404 | user not found | member | 资料查询用户不存在 |

### 5xx 服务端

| code | HTTP | message | 服务 | 场景 |
|------|------|---------|------|------|
| 50001 | 500 / 502 | login failed / proxy build request failed | member / gateway-service | 登录内部错误；网关构建代理请求失败 |
| 50002 | 502 | downstream service unavailable | gateway-service | 下游 member 不可达 |
| 50006 | 500 | logout failed | member | 登出失败 |
| 50007 | 500 | token verify failed | member | introspect 内部错误 |
| 50008 | 500 | register failed | member | 注册内部错误 |
| 50009 | 500 | query user failed | member | 查询资料失败 |
| 50010 | 500 | update profile failed | member | 更新资料失败 |

> **注意**：`50001` 在 gateway-service 表示代理构建失败（502），在 member 表示登录失败（500）。新增码时避免跨服务复用同一数字。

### 501 未实现（WebSocket 占位）

| code | HTTP | message | 服务 | 场景 |
|------|------|---------|------|------|
| 50101 | 501 | websocket not implemented | gateway-service | WS 能力尚未开放 |

## 领域错误映射（member app）

transport 层将 `domain` 包错误映射为上表业务码，例如：

- `ErrInvalidCredentials` → `40103`（登录）
- `ErrUsernameExists` → `40013`
- `ErrTokenInvalid` → `40102`

定义见 `internal/member/domain/types.go`。

## gRPC

gRPC 调用使用标准 `codes.*`（如 `InvalidArgument`、`Unauthenticated`、`Internal`）。`Introspect` 成功时响应消息内仍带 `code: 0`、`message: ok`。详见 [grpc.md](grpc.md)。

## 客户端处理建议

1. `code === 0`：成功，读取 `data`。
2. `40101` / `40102`：引导重新登录。
3. `40xxx`：展示 `message`，修正输入。
4. `500xx` / `502`：可重试；携带 `trace_id` 上报支持。
