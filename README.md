# Go template

Go 最小可运行后端模板，当前只保留 `api-gateway` + `member-service`，用于跑通登录、注册、退出和用户资料接口。

## 服务清单

- `api-gateway` (`:8180`)：统一 HTTP 入口，负责 CORS、鉴权和转发到 member-service。
- `member-service` (`:8181`，gRPC `:9181`)：注册、登录、退出、token introspect、用户资料。

## 中间件

- MySQL：用户数据。
- Redis：token 存储。
- Consul：可选服务注册与发现，默认可通过 `CONSUL_ENABLED=false` 关闭。

## 配置规则

- 环境变量文件：启动时自动读取 `configs/.env`，根目录 `.env` 作为可选兜底。
- 默认环境：`APP_ENV=dev`
- 默认配置文件：`configs/app.<env>.json`
- 手动指定文件：`CONFIG_FILE=...`
- 最终优先级：`系统环境变量 > configs/.env > .env > 配置文件 > 默认值`

配置示例见 `configs/.env.example`。

## 快速启动

```bash
go mod tidy
./scripts/dev-up.sh
curl http://127.0.0.1:8180/healthz
```

## 当前接口

- `POST /v1/auth/register`
- `POST /v1/auth/login`
- `POST /v1/auth/logout`
- `GET /v1/member/users/profile`
- `PUT /v1/member/users/profile`

## 接口测试

Postman/Apifox 可导入 `api/pricing-assistant.postman_collection.json`。

命令行冒烟测试：

```bash
./scripts/smoke-auth-flow.sh
```

可通过环境变量覆盖默认值：

```bash
BASE_URL=http://127.0.0.1:8180 USERNAME=test_user MOBILE=13800138000 ./scripts/smoke-auth-flow.sh
```
