# 新增微服务 Playbook

以新增 **order-service** 为例（仅文档说明，不要求仓库内已存在该服务）。

## 检查清单

- [ ] 复制服务骨架
- [ ] 实现 transport / app / repo
- [ ] Consul 服务名与配置
- [ ] 网关路由表
- [ ] 文档与测试
- [ ] CHANGELOG

## 1. 复制服务骨架

```text
cmd/order-service/main.go          ← 参考 cmd/member-service/main.go
internal/order/
  domain/
  app/
  repo/
  transport/http/handler.go
  vo/
```

模块 import 使用 `go-template/internal/order/...`。

`main.go` 中：

- HTTP 端口从 `cfg.OrderServicePort`（需在 config 扩展）或独立 env 读取
- 注册 Consul 名：**`order-service`**
- 注册 `GET /healthz`

## 2. 实现业务

遵循分层与 [docs/api](../api/README.md)：

- transport 只做校验 + `httpx.JSON` + `errcode` 常量
- app 返回 `domain.Err*`
- 新错误码先登记 [error-codes.md](../api/error-codes.md)

## 3. 扩展配置（工程）

| 文件 | 改动 |
|------|------|
| `internal/platform/config/config.go` | 增加 `OrderServicePort`、`OrderServiceURL` 等 |
| `configs/.env.example` | 增加变量（含 `#` 注释） |
| `configs/.env` | **与 example 同步** |
| `configs/README.md` | 更新 JSON/ENV 字段表 |
| `configs/app.dev.json` / `app.prod.json` | 增加对应 JSON 字段 |

## 4. 网关路由表

编辑 [`internal/gateway/routes/routes.go`](../../internal/gateway/routes/routes.go)：

```go
{
    PublicPath:   "/v1/order/orders",
    UpstreamPath: "/v1/orders",
    ServiceName:  "order-service",
    FallbackURL:  "", // 使用 cfg 注入的 orderServiceURL
    RequiresAuth: true,
},
```

并在 `Handler` 中注入 `orderServiceURL`（与 member 的 `memberURL` 类似），`proxy` 时 `resolve("order-service", fallback)`。

`RequiresAuth` 为 `true` 的路径会自动走网关鉴权，**无需**再改 `RequiresAuth()` 硬编码列表。

本地路由（仅网关自身）在 `RegisterRoutes` 中单独注册，如 `/healthz`、`/v1/ws`。

## 5. 网关 Handler 注入下游地址

`cmd/gateway-service/main.go`：

- 读取 `cfg.OrderServiceURL`
- 传入 `NewHandler(..., orderURL string)` 或扩展 `Handler` 结构体

## 6. Docker / 本地运行

| 文件 | 改动 |
|------|------|
| `Dockerfile` | `go build -o /out/order-service ./cmd/order-service` |
| `docker-compose.yml` | 增加 `order-service` 服务 |
| `scripts/dev-up.sh` | 编译并启动 order（可选） |
| `Makefile` | `build` 目标包含 order |

## 7. 文档与协作

| 文件 | 改动 |
|------|------|
| `docs/api/README.md` | 接口表 |
| `docs/architecture/services.md` | 服务职责 |
| `api/postman_collection.json` | 新接口 |
| `CHANGELOG.md` | `[Unreleased]` |
| `.cursor/skills/api-change/SKILL.md` | 若流程有变则更新 |

## 8. 验证

```bash
make build
make test
make smoke          # 原 member 流程仍通过
curl http://127.0.0.1:8180/v1/order/...   # 新接口
```

## 文件改动汇总（order-service 示例）

| 类型 | 路径 |
|------|------|
| 新建 | `cmd/order-service/main.go` |
| 新建 | `internal/order/**` |
| 修改 | `internal/gateway/transport/http/routes.go` |
| 修改 | `internal/gateway/transport/http/handler.go` |
| 修改 | `cmd/gateway-service/main.go` |
| 修改 | `internal/platform/config/config.go` |
| 修改 | `configs/.env.example`、`configs/.env`、`configs/README.md`、`app.dev.json`、`app.prod.json` |
| 修改 | `docker-compose.yml`、`Dockerfile`、`Makefile` |
| 修改 | `docs/api/*`、`CHANGELOG.md` |
