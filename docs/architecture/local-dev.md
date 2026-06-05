# 本地开发

## 方式一：本机二进制（make dev）

**前提**：本机已运行 MySQL、Redis、RabbitMQ，且 `configs/.env` 中 DSN/Redis/RabbitMQ 正确。

```bash
go mod tidy
make dev          # 等价 ./scripts/dev-up.sh
curl http://127.0.0.1:8180/healthz
make smoke
```

日志：`./logs/gateway-service.log`、`./logs/member-service.log`、`./logs/order-service.log`

## 方式二：Docker Compose（推荐）

一键启动 MySQL、Redis、RabbitMQ、member-service、order-service、gateway-service。

```bash
make compose-up
make smoke
make compose-down
```

Compose 使用 `configs/app.dev.json`，服务间通过 Docker 网络名通信（`mysql`、`redis`、`rabbitmq`、`member-service`、`order-service`）。

### MySQL 初始化

首次启动会将 [`db/go_template_db.sql`](../../db/go_template_db.sql) 导入空库。SQL 含 `DROP TABLE`，**仅适合首次 init**；已有数据卷时请备份或删卷重建。

```bash
docker compose down -v   # 清空数据卷后重新 init
```

## 方式三：单服务调试

仅调试 member 时可直连（勿用于生产对外）：

```bash
go run ./cmd/member-service
curl http://127.0.0.1:8181/healthz
```

## 常用 Make 目标

| 命令 | 说明 |
|------|------|
| `make build` | 编译三个服务到 `./bin/` |
| `make dev` | 本机启动 |
| `make compose-up` | Docker Compose 后台启动 |
| `make compose-down` | 停止并移除容器 |
| `make proto` | 生成 gRPC 代码 |
| `make test` | 运行单元测试 |
| `make smoke` | 认证 + 订单 Demo 冒烟 |

## 合并前自检

```bash
make test
make smoke
```

GitHub Actions **默认不自动运行**（避免每次 push 触发）；需在仓库 **Actions → CI → Run workflow** 手动执行。合并前建议本地先跑 `make test && make smoke`。若需 push/PR 自动 CI，见 [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) 顶部注释。
