# 配置说明

## 加载顺序

1. 代码 `defaultConfig()`
2. `configs/.env`（不覆盖已存在系统环境变量）
3. `.env`（可选）
4. `configs/app.<APP_ENV>.json` 或 `CONFIG_FILE`
5. 系统环境变量再次覆盖

实现：[`internal/platform/config/config.go`](../../internal/platform/config/config.go)

## 环境变量（configs/.env）

| 变量 | 说明 | 默认 |
|------|------|------|
| `GATEWAY_SERVICE_PORT` | 网关 HTTP 端口 | 8180 |
| `MEMBER_SERVICE_PORT` | member HTTP 端口 | 8181 |
| `MEMBER_SERVICE_GRPC_PORT` | member gRPC 端口 | 9181 |
| `APP_ENV` | 环境名，决定 `app.<env>.json` | dev |
| `CONFIG_FILE` | 手动指定 JSON 配置路径 | — |
| `MEMBER_SERVICE_URL` | 网关回退 member HTTP 基址 | http://127.0.0.1:8181 |
| `MEMBER_SERVICE_GRPC_ADDR` | 网关 gRPC 地址 | 127.0.0.1:9181 |
| `GATEWAY_USE_MEMBER_GRPC` | 鉴权是否优先 gRPC | false |
| `MYSQL_DSN` | MySQL 连接串 | — |
| `REDIS_ADDR` | Redis 地址 | — |
| `REDIS_PASSWORD` | Redis 密码 | — |
| `REDIS_DB` | Redis DB 序号 | 0 |
| `CONSUL_ENABLED` | 是否启用 Consul | false |
| `CONSUL_ADDRESS` | Consul 地址 | 127.0.0.1:8500 |
| `CONSUL_DATACENTER` | Consul DC | dc1 |
| `SERVICE_HOST` | 注册到 Consul 的 host | 127.0.0.1 |
| `CORS_ALLOW_ORIGIN` | 跨域 Allow-Origin | http://localhost:5173 |

模板 **仅包含代码会读取的变量**；勿在 `.env` 添加未实现功能的键。

## JSON 配置文件

| 文件 | 场景 |
|------|------|
| `configs/app.dev.json` | 本地 / docker-compose（服务名 `mysql`、`redis`） |
| `configs/app.prod.json` | 生产 / K8s（服务名 `member-service` 等） |

字段与结构体 `json` tag 对应，例如 `gateway_service_port`、`member_service_url`。

## dev vs prod 差异

| 项 | dev | prod |
|----|-----|------|
| member URL | `http://127.0.0.1:8181` 或 `http://member-service:8181` | `http://member-service:8181` |
| MySQL host | `127.0.0.1` / `mysql` | `mysql` |
| Consul | 通常关闭 | 按需开启 |
| CORS | `localhost:5173` | 正式前端域名 |

## 可选基础设施（v0.3）

`scripts/rabbitmq-*.command` 等脚本为本地辅助工具，**当前 Go 代码未读取 RabbitMQ 环境变量**，接入计划见 README 路线图 v0.3。
