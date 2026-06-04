# WebSocket 实时通道

> **状态**：规范已定义；Gateway 提供 **占位路由**，完整 Hub 与推送能力在后续迭代实现。

## 目标

- 统一由 **gateway-service** 暴露 WebSocket 入口，与 REST 共用鉴权与 trace 体系。
- 帧结构与 REST 信封一致，降低前后端认知成本。

## 入口（规划）

| 路径 | 说明 |
|------|------|
| `GET /v1/ws` | 主入口（占位已实现，返回 501） |

握手前鉴权与 HTTP 相同：`Authorization: Bearer <token>` 或 `token` 头；不得在 URL 长期携带 token（仅调试可用 query）。

## 帧格式（规划）

```json
{
  "type": "event",
  "code": 0,
  "message": "ok",
  "data": {},
  "trace_id": "..."
}
```

| type | 方向 | 说明 |
|------|------|------|
| `req` | 客户端 → 服务端 | 请求 |
| `ack` | 服务端 → 客户端 | 请求响应 |
| `event` | 服务端 → 客户端 | 服务端推送 |
| `error` | 双向 | 错误；`code` 非 0 |

### 心跳（规划）

客户端周期性发送：

```json
{ "type": "req", "code": 0, "message": "ping", "data": {} }
```

服务端 `ack`：`message: "pong"`。

## 主题命名（规划）

格式：`{domain}.{action}`，例如：

- `user.profile_updated`
- `notify.message`

订阅与权限校验在 Hub 层按 userID 隔离（待实现）。

## 错误码

与 HTTP 共用段规则，WebSocket 占位专用：

| code | HTTP（占位） | message | 说明 |
|------|----------------|---------|------|
| 50101 | 501 | websocket not implemented | 功能未开放 |

鉴权类错误在连接建立前仍用 HTTP 401 + `40101`/`40102`（与 REST 一致）。

## 实现路线图

| 阶段 | 内容 |
|------|------|
| 当前 | `internal/platform/ws` Hub 接口 + Noop；Gateway `/v1/ws` 返回 501 + `50101` |
| 下一迭代 | 引入 `gorilla/websocket` 或标准库升级；握手鉴权、连接管理 |
| 后续 | 主题订阅、与 Redis/pub-sub 或 MQ 集成 |

## 客户端建议

在收到 `50101` 前勿依赖 WS；实时需求可临时使用轮询或 SSE（若后续增加）。
