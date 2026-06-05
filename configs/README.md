# 配置文件说明

本目录存放环境变量与 JSON 配置。**JSON 标准不支持注释**，字段说明见下文；带 `#` 注释的变量见 [`.env.example`](.env.example)。

## 文件清单

| 文件 | 用途 | 是否入库 |
|------|------|----------|
| `.env.example` | 环境变量模板（含注释） | 是 |
| `.env` | 本地实际环境变量（与 example 保持同步） | 是（仅 dev 占位值） |
| `app.dev.json` | Docker Compose / 本地容器网络 | 是 |
| `app.prod.json` | 生产 / K8s 示例 | 是 |

## 维护约定

1. **新增或修改环境变量**：先改 `.env.example`（含注释），再 **同步** `.env` 中对应键值。
2. **新增 JSON 字段**：同步改 `app.dev.json`、`app.prod.json`，并更新本文档与 [`docs/architecture/config.md`](../docs/architecture/config.md)。
3. Agent/开发者流程见 `.cursor/rules/config-env.mdc`、`.cursor/skills/code-change/SKILL.md`。

加载优先级：`系统环境变量` > `configs/.env` > 根目录 `.env` > `configs/app.<APP_ENV>.json` > 代码默认值。

---

## 环境变量（.env）

与 [`.env.example`](.env.example) 一一对应，完整表见 [`docs/architecture/config.md`](../docs/architecture/config.md)。

---

## JSON 字段（app.dev.json / app.prod.json）

| JSON 字段 | 说明 | dev 示例 | prod 注意 |
|-----------|------|----------|-----------|
| `gateway_service_port` | 网关 HTTP 端口 | `8180` | 与 Ingress 一致 |
| `member_service_port` | member HTTP 端口 | `8181` | 内网 |
| `member_service_grpc_port` | member gRPC 端口 | `9181` | 内网 |
| `member_service_url` | 网关回退 member HTTP | `http://member-service:8181` | 服务名或 K8s DNS |
| `member_service_grpc_addr` | 网关 gRPC 地址 | `member-service:9181` | 同上 |
| `gateway_use_member_grpc` | 鉴权优先 gRPC | `false` | 按延迟选型 |
| `order_service_port` | order HTTP 端口 | `8182` | 内网 |
| `order_service_url` | 网关回退 order HTTP | `http://order-service:8182` | 服务名 |
| `mysql_dsn` | MySQL 连接串 | root@mysql:3306 | **生产换强密码** |
| `redis_addr` | Redis 地址 | `redis:6379` | 内网 |
| `redis_password` | Redis 密码 | `redis1234` | 生产必填 |
| `redis_db` | Redis DB 序号 | `0` | — |
| `mq_provider` | `rabbitmq` / `rocketmq` | `rabbitmq` | 与基础设施一致 |
| `mq_auto_declare` | Rabbit 自动声明队列 | `true` | 生产建议 `false` |
| `rabbitmq_url` | AMQP 连接串 | `amqp://guest:guest@rabbitmq:5672/` | provider=rabbitmq 时必填 |
| `rocketmq_namesrv` | NameServer | `127.0.0.1:9876` | provider=rocketmq 时必填 |
| `consul_enabled` | 是否启用 Consul | `false` | 按需 |
| `consul_address` | Consul 地址 | `consul:8500` | — |
| `consul_datacenter` | Consul DC | `dc1` | — |
| `service_host` | 注册 Consul 的 host | `member-service` | 容器填服务名 |
| `cors_allow_origin` | CORS Allow-Origin | `http://localhost:5173` | 正式前端域名 |

### dev vs prod 差异摘要

- **dev**：`mq_auto_declare=true`，MySQL/Redis 使用 compose 默认密码。
- **prod**：`mq_auto_declare=false`（运维预建 MQ 资源），`cors_allow_origin` 为正式域名，DSN/密码走密钥管理。

MQ 详细说明：[`docs/architecture/mq.md`](../docs/architecture/mq.md)。
