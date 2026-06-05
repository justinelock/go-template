# 消息队列（MQ）使用指南

本模版通过 **`internal/platform/mq.Bus`** 统一消息发布与订阅。业务代码只使用 **逻辑 Topic / Group**，通过配置 **`MQ_PROVIDER`** 在 **RabbitMQ（默认）** 与 **RocketMQ** 之间切换，无需改动 order-service 业务逻辑。

---

## 1. 架构概览

```mermaid
flowchart TB
  subgraph orderService [order-service]
    App[app.CreateOrder]
    PubAdapter[order/mq.SettlePublisher]
    Worker[worker.StartSettlement]
  end

  Factory["mq.New(cfg)"]
  Bus[mq.Bus 接口]
  Rabbit[rabbitBus]
  Rocket[rocketBus]

  App --> PubAdapter
  PubAdapter --> Bus
  Worker --> Bus
  Factory --> Bus
  Bus --> Rabbit
  Bus --> Rocket
```

| 层级 | 路径 | 职责 |
|------|------|------|
| 平台 | `internal/platform/mq` | `Bus`、`Message`、工厂、Rabbit/Rocket 实现 |
| 适配 | `internal/order/mq` | 将 `Bus` 适配为 `app.Publisher` |
| 业务 | `internal/order/app` | 只依赖 `Publisher` 接口 |
| 消费 | `internal/order/worker` | `bus.Subscribe(TopicOrderSettle, ...)` |

---

## 2. 概念对照：RabbitMQ vs RocketMQ

业务与代码 **只使用左侧「逻辑名」**；右侧为各 MQ 的物理资源，在 [`topics.go`](../../internal/platform/mq/topics.go) 集中映射。

| 逻辑（统一） | RabbitMQ | RocketMQ |
|--------------|----------|----------|
| Topic `order.settle` | Exchange `go_template.demo` + RoutingKey `order.settle` + Queue `order.settle` | Topic `order_settle` |
| Group `order-settlement` | Consumer tag（绑定到 queue） | **ConsumerGroup** `order-settlement` |

说明：

- RabbitMQ 用 **Exchange → Queue → Binding**；RocketMQ 用 **Topic + ConsumerGroup**（无 Exchange）。
- RocketMQ 的「Queue」是 Topic 内部分区，由 Broker 管理，**不要**与 Rabbit 的 Queue 名一一对应。

---

## 3. 配置与切换

实现：[`internal/platform/config/config.go`](../../internal/platform/config/config.go)

| 变量 / JSON 字段 | 默认 | 说明 |
|------------------|------|------|
| `MQ_PROVIDER` / `mq_provider` | `rabbitmq` | `rabbitmq` 或 `rocketmq` |
| `MQ_AUTO_DECLARE` / `mq_auto_declare` | `true` | Rabbit：启动时声明 exchange/queue；Rocket：依赖 Broker `autoCreateTopicEnable` |
| `RABBITMQ_URL` / `rabbitmq_url` | `amqp://guest:guest@127.0.0.1:5672/` | **rabbitmq** 时必填 |
| `ROCKETMQ_NAMESRV` / `rocketmq_namesrv` | `127.0.0.1:9876` | **rocketmq** 时必填 |

### 3.1 使用 RabbitMQ（默认）

`configs/app.dev.json`：

```json
{
  "mq_provider": "rabbitmq",
  "mq_auto_declare": true,
  "rabbitmq_url": "amqp://guest:guest@rabbitmq:5672/"
}
```

或环境变量：

```bash
MQ_PROVIDER=rabbitmq
RABBITMQ_URL=amqp://guest:guest@127.0.0.1:5672/
```

Docker Compose 已内置 `rabbitmq` 服务，`make compose-up` 即可。

### 3.2 切换到 RocketMQ

```bash
MQ_PROVIDER=rocketmq
ROCKETMQ_NAMESRV=127.0.0.1:9876
# 本机或云托管 NameServer 地址，可多个用分号分隔（客户端 API 传数组）
```

**注意**：当前 `docker-compose.yml` **不包含** RocketMQ 容器；需本机安装、云 EMQ 或自建集群。详见下文 §5。

### 3.3 生产建议

| 项 | dev | prod |
|----|-----|------|
| `mq_auto_declare` | `true` | `false`（运维预先建好资源） |
| 连接地址 | `rabbitmq:5672` | 托管 MQ 内网地址 |

生产关闭自动声明时，须先 **手动创建** §4 / §5 中的物理资源，否则启动或消费会失败。

---

## 4. RabbitMQ 接入

### 4.1 Compose 一键（推荐本地）

```bash
make compose-up
make smoke
```

- AMQP：`5672`
- 管理台：http://127.0.0.1:15672（`guest` / `guest`）

### 4.2 自动声明（`mq_auto_declare=true`）

`order-service` 启动时 `mq.New(cfg)` 会创建：

| 资源 | 名称 | 类型 |
|------|------|------|
| Exchange | `go_template.demo` | direct, durable |
| Queue | `order.settle` | durable |
| Binding | routing key `order.settle` → queue `order.settle` | — |

代码：[`internal/platform/mq/rabbitmq.go`](../../internal/platform/mq/rabbitmq.go) 的 `declareKnownTopics`。

### 4.3 手动创建（生产或关闭 auto_declare）

**管理台**：Queues → Add a new queue → `order.settle`；Exchanges → `go_template.demo`；Bindings 绑定 routing key `order.settle`。

**CLI**（需安装 `rabbitmqadmin`）：

```bash
rabbitmqadmin declare exchange name=go_template.demo type=direct durable=true
rabbitmqadmin declare queue name=order.settle durable=true
rabbitmqadmin declare binding source=go_template.demo destination=order.settle routing_key=order.settle
```

### 4.4 验证

```bash
# 发布测试（需 order-service 运行中下单）
curl -s http://127.0.0.1:15672/api/queues/%2F/order.settle -u guest:guest | python3 -m json.tool
```

---

## 5. RocketMQ 接入

Compose **未** 包含 RocketMQ；按以下方式自建后切换 `MQ_PROVIDER=rocketmq`。

### 5.1 本机快速启动（示例）

使用 Apache 官方镜像（仅作本地验证参考）：

```bash
# NameServer
docker run -d --name rmqnamesrv -p 9876:9876 apache/rocketmq:5.3.2 sh mqnamesrv

# Broker（需能访问 namesrv）
docker run -d --name rmqbroker -p 10911:10911 -p 10909:10909 \
  -e NAMESRV_ADDR=host.docker.internal:9876 \
  apache/rocketmq:5.3.2 sh mqbroker -n host.docker.internal:9876
```

云环境使用阿里云 **消息队列 RocketMQ 版** 时，将控制台中的 **NameServer 地址** 填入 `ROCKETMQ_NAMESRV`。

### 5.2 手动创建 Topic 与 ConsumerGroup

| 逻辑 | RocketMQ 物理名 |
|------|-----------------|
| Topic `order.settle` | Topic **`order_settle`** |
| Group `order-settlement` | ConsumerGroup **`order-settlement`** |

**mqadmin**（在 Broker 容器内或安装目录下）：

```bash
sh mqadmin updateTopic -n 127.0.0.1:9876 -t order_settle -c DefaultCluster
sh mqadmin updateSubGroup -n 127.0.0.1:9876 -g order-settlement -c DefaultCluster
```

dev 若 Broker 开启 `autoCreateTopicEnable=true`，首次发布可能自动建 Topic；**ConsumerGroup 仍建议显式创建**。

### 5.3 启动 order-service

```bash
export MQ_PROVIDER=rocketmq
export ROCKETMQ_NAMESRV=127.0.0.1:9876
export MYSQL_DSN=...
export REDIS_ADDR=...
go run ./cmd/order-service
```

日志应出现：`mq provider=rocketmq`。

---

## 6. 业务如何使用（order Demo）

### 6.1 发布链路

```mermaid
sequenceDiagram
  participant Client
  participant Gateway
  participant OrderHTTP
  participant App as app.Service
  participant Pub as SettlePublisher
  participant Bus as mq.Bus
  participant MQ as RabbitMQ_or_RocketMQ

  Client->>Gateway: POST /v1/order/orders
  Gateway->>OrderHTTP: X-User-Id + X-Idempotency-Key
  OrderHTTP->>App: CreateOrder
  App->>App: Redis幂等 + 锁 + MySQL INSERT
  App->>Pub: PublishSettle(orderID)
  Pub->>Bus: Publish Topic order.settle
  Bus->>MQ: 物理投递
  App-->>Client: orderId pending
```

关键代码：

- 发布接口：[`internal/order/app/service.go`](../../internal/order/app/service.go) → `publisher.PublishSettle`
- 适配器：[`internal/order/mq/publisher.go`](../../internal/order/mq/publisher.go)
- 组装：[`cmd/order-service/main.go`](../../cmd/order-service/main.go) → `mq.New(cfg)` + `ordermq.NewSettlePublisher(bus)`

### 6.2 消费链路

- Worker：[`internal/order/worker/settlement.go`](../../internal/order/worker/settlement.go)
- 订阅：`bus.Subscribe(ctx, mq.TopicOrderSettle, mq.GroupOrderSettlement, handler)`
- 处理：`svc.SettleOrder` 将 MySQL 中订单标为 `settled`

### 6.3 curl 验证（RabbitMQ + Compose）

```bash
make compose-up

# 注册登录
BASE=http://127.0.0.1:8180
# ... 见 order-demo.md 或：
make smoke
```

期望：下单后 `status=pending`，数秒内变为 `settled`；相同 `X-Idempotency-Key` 重复 POST 返回同一 `orderId`。

### 6.4 业务侧要写什么？

**不需要** import `amqp` 或 `rocketmq`。新增「发消息」时：

1. 在 [`topics.go`](../../internal/platform/mq/topics.go) 增加逻辑 Topic 与 Rabbit/Rocket 映射
2. 在 `internal/<svc>/mq/` 增加 Publisher 适配器（或直接用 `bus.Publish`）
3. 在 worker 或独立 consumer 中 `bus.Subscribe`
4. 按 §4 / §5 手动或自动创建物理资源
5. 更新本文档映射表

---

## 7. 平台 API 参考

### 7.1 Bus 接口

```go
type Message struct {
    Topic   string            // 逻辑 Topic，如 order.settle
    Key     string            // 顺序/排查键（Rocket MessageKey）
    Body    []byte
    Headers map[string]string
}

type Bus interface {
    Publish(ctx context.Context, msg Message) error
    Subscribe(ctx context.Context, topic, group string, handler Handler) error
    Close() error
}
```

### 7.2 工厂

```go
bus, err := mq.New(cfg)
defer bus.Close()
```

### 7.3 逻辑常量

```go
mq.TopicOrderSettle     // "order.settle"
mq.GroupOrderSettlement // "order-settlement"
```

---

## 8. 新增业务 Topic 检查清单

- [ ] 在 `topics.go` 增加逻辑 Topic 常量
- [ ] 填写 `rabbitMappings` 与 `rocketTopicNames`
- [ ] 文档更新 §2 映射表
- [ ] Rabbit：exchange/queue/binding 或依赖 `mq_auto_declare`
- [ ] Rocket：Topic + ConsumerGroup（mqadmin 或控制台）
- [ ] 业务 Publisher / Subscribe 使用逻辑名
- [ ] `CHANGELOG.md` 记录

---

## 9. 故障排查

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| `mq init failed: rabbitmq_url is required` | provider=rabbitmq 但未配 URL | 设置 `RABBITMQ_URL` |
| `mq init failed: rocketmq_namesrv is required` | provider=rocketmq 但未配 NameSrv | 设置 `ROCKETMQ_NAMESRV` |
| `unsupported mq_provider` | 拼写错误 | 仅 `rabbitmq` / `rocketmq` |
| 订单一直 pending | 消费者未启动或 MQ 不可达 | 查 order-service 日志、`make smoke` |
| Rabbit 管理台无队列 | auto_declare 关闭且未手动建 | §4.3 手动创建 |
| Rocket 发送失败 NO_TOPIC | Topic 未创建且 Broker 禁止自动建 | §5.2 创建 `order_settle` |
| 重复消费 | 至少一次语义 | 消费端幂等（订单已 settled 可忽略） |

---

## 10. 相关文档

- [order-demo.md](order-demo.md) — 订单全流程 Demo
- [redis-high-concurrency.md](redis-high-concurrency.md) — Redis 幂等/锁（与 MQ 配合）
- [config.md](config.md) — 全部环境变量
- [add-service.md](add-service.md) — 新服务接入 playbook
