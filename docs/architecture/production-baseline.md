# 中小型生产基线清单（v0.4–v0.5）

## 已具备

| 类别 | 能力 |
|------|------|
| 可观测 | slog 日志、`/readyz`、`/metrics`、可选 OTel+Jaeger |
| 治理 | 网关限流、代理超时、断路器、优雅关闭、panic recovery |
| 安全 | bcrypt、refresh token、简易 RBAC |
| 业务域 | gateway + member + order + **payment** Demo |

## 与重型方案对比（刻意未引入）

- 配置中心（Nacos/Apollo）、Vault、服务网格、mTLS 全链路、Casbin

## 合并前自检

```bash
make test && make smoke
curl -s http://127.0.0.1:8180/readyz
```

可选：`docker compose --profile obs up -d` 验证 Grafana/Jaeger。
