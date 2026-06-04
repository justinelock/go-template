# REST 约定

## 版本与路径

- 所有对外业务 API 使用前缀 **`/v1/`**。
- 路径风格：小写、分段用 `/`，资源名复数或领域前缀，例如 `/v1/auth/login`、`/v1/member/users/profile`。
- **网关路径映射**：对外 `/v1/member/...` 可映射到 member 内部路径（当前：`/v1/member/users/profile` → member `/v1/users/profile`）。

新增领域时建议：`/v1/{domain}/...`（如 `member`、`order`），由 gateway 统一暴露，避免客户端直连各微服务。

## HTTP 方法

| 方法 | 用途 |
|------|------|
| GET | 查询 |
| POST | 创建、登录、登出等非幂等写操作 |
| PUT | 全量或部分更新（当前资料接口用 PUT） |
| DELETE | 删除（预留） |
| OPTIONS | CORS 预检，网关返回 `204` |

方法不允许时返回 **HTTP 405**，业务码 **`405`**，`message`: `method not allowed`。

## 响应信封

所有由本模板服务**直接生成**的 JSON 响应（gateway 自身错误、member HTTP handler）使用统一结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "trace_id": "a1b2c3d4e5f67890"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 业务码，`0` 表示成功 |
| `message` | string | 人类可读说明 |
| `data` | object / array / null | 成功时载荷；失败常为 `null` 或省略 |
| `trace_id` | string | 链路 ID，与响应头 `X-Trace-Id` 一致 |

实现：`internal/platform/httpx/httpx.go`。

### HTTP 状态码 vs 业务码

二者**分离**：例如参数错误为 HTTP **400** + 业务码 **40001**。客户端应以 **`code`** 作为业务分支依据，`HTTP status` 仅作传输层分类。

### 网关代理成功

下游 member 返回的 status 与 body **原样透传**，网关不再包一层信封。仅网关构建请求失败、下游不可达、网关鉴权失败时由网关用 `httpx.JSON` 写信封。

## 链路追踪

- 请求头 **`X-Trace-Id`**：客户端可传入；缺失时服务端生成 16 位 hex。
- 响应头 **`X-Trace-Id`**：与 body 中 `trace_id` 一致。
- gRPC introspect 请求体字段 `trace_id` 与 HTTP 对齐。

## 请求头

| 头 | 说明 |
|----|------|
| `Content-Type` | JSON 请求使用 `application/json` |
| `Authorization` | `Bearer <token>`，推荐 |
| `token` | 与 Bearer 二选一或并存（兼容） |
| `X-Trace-Id` | 可选，链路 ID |
| `X-Idempotency-Key` | 可选，写操作幂等（网关会透传给下游） |
| `X-User-Id` | **仅服务端**：网关鉴权后注入，member 可信任此头解析用户 |

CORS 暴露头：`X-Trace-Id`（见 gateway `withCORS`）。

## JSON 字段命名

- 对外请求/响应以 **camelCase** 为主：`parentId`、`inviteCode`、`accessToken`。
- 兼容旧客户端时，可在同一对象中保留 snake_case 副本（如登出响应 `loggedOut` + `logged_out`；introspect `userId` + `user_id`）。
- 新增字段优先 camelCase；废弃 snake_case 需在文档标注版本。

## 分层与新增接口流程

```
cmd/{service}/main.go
  → internal/{service}/transport/http/handler.go   # 路由、参数校验、错误码映射
  → internal/{service}/app/                        # 用例与领域规则
  → internal/{service}/repo/                       # 持久化
```

- transport 层：解析 body、校验 method、调用 `app`，用 `httpx.JSON` 输出。
- app 层：返回 `domain` 错误变量，**不**写 HTTP 状态码。
- gateway transport：**禁止**业务逻辑，仅鉴权、代理、健康检查、公开配置。

## 健康与公开接口

| 路径 | 说明 |
|------|------|
| `GET /healthz` | 服务存活，`data.service` 标识服务名 |
| `GET /v1/public/config` | 前端启动用公开配置（无敏感信息） |
