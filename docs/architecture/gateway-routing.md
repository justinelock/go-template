# 网关路由与热加载

网关的反向代理路由表支持**运行时热加载**：改路由无需重新编译、无需重启进程。

## 设计要点

- **动态分发**：标准库 `http.ServeMux` 无法在运行时增删已注册路由，因此业务代理路由不再逐条注册，而是统一挂到 catch-all `"/"`，请求时查 `routes` 包的**原子快照**（`atomic.Pointer`）分发。健康检查 `/healthz`、`/readyz`、`/metrics`、`/v1/public/config`、`/v1/ws` 仍是显式路由，优先级高于 catch-all。
- **开箱即用**：`configs/routes.json` 缺失或损坏时，回退到代码内置默认路由（`internal/gateway/routes/routes.go` 的 `builtinTable`），零配置即可跑通。
- **原子热替换**：新表校验通过后整体 `Store`，in-flight 请求要么读到旧表要么读到新表，不会读到半更新状态。
- **坏配置不打挂网关**：任一条目非法 → 整份拒绝并保留上一版路由，仅记录 `error` 日志。
- **零第三方依赖**：热加载靠「定时轮询文件 mtime」+「`SIGHUP` 信号」双触发，不引入 fsnotify 等依赖。

## 配置文件 `configs/routes.json`

```json
{
  "routes": [
    {"public_path": "/v1/auth/login", "upstream_path": "/v1/auth/login", "service_name": "member-service", "requires_auth": false},
    {"public_path": "/v1/order/orders/", "upstream_path": "/v1/orders/", "service_name": "order-service", "requires_auth": true, "match": "prefix"},
    {"public_path": "/v1/foo", "upstream_path": "/foo", "upstream_base_url": "http://foo-service:8190", "requires_auth": true}
  ]
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `public_path` | 是 | 网关对外路径，须以 `/` 开头；`match=prefix` 时须以 `/` 结尾 |
| `upstream_path` | 是 | 下游内部路径，须以 `/` 开头 |
| `service_name` | 二选一 | 服务发现用服务名（与 `upstream_base_url` 至少填一个）|
| `upstream_base_url` | 二选一 | 直连下游基址（形如 `http://host:port`）。**填了它即可纯配置接入新服务，无需改代码** |
| `requires_auth` | 否 | 是否在网关层校验 token，默认 `false` |
| `required_roles` | 否 | 非空时要求用户角色命中其一（RBAC）|
| `match` | 否 | `exact`（默认，精确匹配）或 `prefix`（前缀匹配，如 `/{id}`）|

前缀路由按 `public_path` 长度降序匹配，更具体的前缀优先命中。

## 相关配置项

| 配置键 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `routes_config_path` | `ROUTES_CONFIG_PATH` | `configs/routes.json` | 路由文件路径；留空则仅用内置默认路由 |
| `routes_reload_sec` | `ROUTES_RELOAD_SEC` | `10` | 轮询间隔秒数；`<=0` 时仅靠 `SIGHUP` 触发 |

## 触发热加载

- **改文件**：编辑 `configs/routes.json` 保存，最长 `routes_reload_sec` 秒后自动生效。
- **发信号**：`kill -HUP <gateway-pid>` 立即重载（容器内 `kill -HUP 1`）。

重载成功/失败都会打印结构化日志：

```
INFO gateway routes: reloaded path=configs/routes.json count=9
ERROR gateway routes: reload failed, keeping previous table path=... err=...
```
