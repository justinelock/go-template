# 架构总览

go-template 是微服务模版：**gateway-service** 对外，**member-service** 承载用户域，**order-service** 为订单高并发 Demo。

## 拓扑

```mermaid
flowchart LR
  Client --> Gateway[gateway_service_8180]
  Gateway -->|HTTP proxy| MemberHTTP[member_http_8181]
  Gateway -->|HTTP proxy| OrderHTTP[order_http_8182]
  Gateway -->|gRPC introspect| MemberGRPC[member_grpc_9181]
  MemberHTTP --> MySQL[(MySQL)]
  MemberHTTP --> Redis[(Redis)]
  OrderHTTP --> MySQL
  OrderHTTP --> Redis
  OrderHTTP --> RabbitMQ[(RabbitMQ)]
  Gateway -.->|optional| Consul[Consul]
```

## 端口

| 服务 | HTTP | gRPC |
|------|------|------|
| gateway-service | 8180 | — |
| member-service | 8181 | 9181 |
| order-service | 8182 | — |

## 依赖

| 组件 | 用途 | 必需 |
|------|------|------|
| MySQL | 用户表、订单表 | 是 |
| Redis | token、幂等、分布式锁 | 是 |
| RabbitMQ | 订单异步结算 Demo | order-service 需要 |
| Consul | 服务发现 | 否 |

## 配置优先级

`系统环境变量` > `configs/.env` > `.env` > `configs/app.<env>.json` > 代码默认值

详见 [config.md](config.md)。

## 文档导航

| 文档 | 内容 |
|------|------|
| [services.md](services.md) | 服务职责与禁止事项 |
| [config.md](config.md) | 配置字段表 |
| [local-dev.md](local-dev.md) | 本机 / Compose 启动 |
| [add-service.md](add-service.md) | **新增微服务 playbook** |
| [order-demo.md](order-demo.md) | 订单 Demo（Redis/MQ 高并发示例） |
| [redis-high-concurrency.md](redis-high-concurrency.md) | **Redis 高并发**：分布式锁、幂等、场景示例 |

## 对外原则

- 浏览器与第三方 **只访问 gateway-service**
- member-service HTTP 仅 dev 调试或内网
- 网关 **不写业务逻辑**（鉴权、CORS、代理除外）
