# 安全基线（SMB）

## 认证

- 不透明 Redis access token（非 JWT），TTL 12h
- `POST /v1/auth/refresh` 使用 refresh token（Redis TTL 7d）
- 密码：新用户使用 bcrypt；旧 MD5 登录后懒升级

## RBAC

- `users.role`：`user`（默认）| `admin`
- 网关路由表 `RequiredRoles` 可选校验；网关注入 `X-User-Role`

## Secrets

- 禁止将生产密码提交仓库；使用环境变量或 K8s Secret
- 参考 [`configs/.env.example`](../../configs/.env.example)

## TLS

- 生产建议在 Ingress/Nginx 终止 TLS；gRPC 生产换 `credentials.NewTLS`（开发默认 insecure）

## 限流

- 网关 `RATE_LIMIT_ENABLED=true` 时按 IP+路由前缀 Redis 限流
