# go-template

Go **微服务框架模版**：`gateway-service` + `member-service` + `order-service` + `payment-service` 骨架，含 SMB 生产基线（日志/指标/就绪探针/限流/安全），fork 后可按 playbook 扩展业务服务。

## 适用场景

- 新项目快速搭通「网关 + 用户域 + MySQL/Redis」
- 团队统一 REST 信封、错误码、注释与变更日志
- 培训微服务分层、网关代理、gRPC introspect 回退

## 服务清单

- `gateway-service` (`:8180`)：统一 HTTP 入口，CORS、鉴权、反向代理（无业务逻辑）。
- `member-service` (`:8181`，gRPC `:9181`)：注册、登录、登出、token introspect、用户资料。
- `order-service` (`:8182`)：订单 Demo（Redis 幂等/锁 + MQ 支付/结算），见 [order-demo.md](docs/architecture/order-demo.md)。
- `payment-service` (`:8183`)：支付 Demo（mock-pay + 回调占位），见 [payment-demo.md](docs/architecture/payment-demo.md)。

## 目录结构

```
cmd/
  gateway-service/     # 网关入口
  member-service/      # 业务服务入口
  order-service/       # 订单 Demo 入口
  payment-service/     # 支付 Demo 入口
internal/
  gateway/             # 鉴权、代理、路由表
  member/              # transport → app → repo → domain
  order/               # 订单 Demo 域
  platform/            # config、httpx、errcode、discovery、redislock、idempotency、mq、ws
api/
  proto/               # gRPC 定义
  gen/                 # protoc 生成（勿手改）
docs/
  architecture/        # 拓扑、配置、本地开发、加服务 playbook
  api/                 # REST/gRPC/WS 规范
configs/               # .env、app.<env>.json
scripts/               # dev-up、smoke、gen-proto
```

## 中间件

- MySQL：用户与订单数据
- Redis：token 存储、幂等键、分布式锁
- MQ：订单异步结算（**RabbitMQ 默认**，`MQ_PROVIDER` 可切 RocketMQ，见 [mq.md](docs/architecture/mq.md)）
- Consul：可选服务注册与发现（`CONSUL_ENABLED=false` 可关闭）

## 配置规则

- 启动时读取 `configs/.env`，根目录 `.env` 可选兜底
- 默认环境：`APP_ENV=dev`，配置文件 `configs/app.<env>.json`
- 手动指定：`CONFIG_FILE=...`
- 优先级：**系统环境变量 > configs/.env > .env > 配置文件 > 默认值**

示例见 [`configs/.env.example`](configs/.env.example)。

## 快速启动

### 本机二进制（需本地 MySQL/Redis）

```bash
go mod tidy
make dev
curl http://127.0.0.1:8180/healthz
```

### Docker Compose（推荐新人）

```bash
make compose-up
make smoke
```

详见 [docs/architecture/local-dev.md](docs/architecture/local-dev.md)。

## 当前接口

- `POST /v1/auth/register`
- `POST /v1/auth/login`
- `POST /v1/auth/logout`
- `POST /v1/auth/refresh`
- `GET /v1/member/users/profile`
- `PUT /v1/member/users/profile`
- `POST /v1/order/orders`（需鉴权 + `X-Idempotency-Key`）
- `GET /v1/order/orders/{id}`（需鉴权）
- `POST /v1/payment/payments`、`POST /v1/payment/payments/{id}/mock-pay`（需鉴权，mock 仅 dev）

完整约定见 [docs/api/README.md](docs/api/README.md)。生产基线见 [production-baseline.md](docs/architecture/production-baseline.md)。

## 文档索引

| 文档 | 说明 |
|------|------|
| [docs/architecture/](docs/architecture/README.md) | 架构、配置、本地开发、**Redis/MQ**、**新增服务 playbook** |
| [docs/api/](docs/api/README.md) | API 规范与错误码 |
| [AGENTS.md](AGENTS.md) | Cursor Agent 指引 |
| [CHANGELOG.md](CHANGELOG.md) | 变更日志 |

## 接口测试

Postman/Apifox：`api/postman_collection.json`

```bash
make smoke
# 或
BASE_URL=http://127.0.0.1:8180 ./scripts/smoke-auth-flow.sh
```

合并前自检：`make test && make smoke`（GitHub CI 默认关闭自动运行，需 Actions 页手动 Run workflow）

## 路线图

| 版本 | 内容 |
|------|------|
| **v0.2** | 架构文档、docker-compose、errcode、网关路由表、单元测试 |
| **v0.3** | order-service Demo、RabbitMQ、CI（默认手动触发） |
| **v0.4–v0.5** | 可观测/治理/安全基线、payment-service、OpenAPI 骨架 |
