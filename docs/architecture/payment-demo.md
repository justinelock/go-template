# payment-service Demo

## 流程

```text
下单 → payment.created(MQ) → payment-service 建单
     → mock-pay / 渠道回调 → payment.paid(MQ)
     → order-service 标记 paid → order.settle → settled
```

## 接口（经网关）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/payment/payments` | 创建支付单 |
| GET | `/v1/payment/payments/{id}` | 查询 |
| POST | `/v1/payment/payments/{id}/mock-pay` | 仅 dev（`PAYMENT_MOCK_PAY_ENABLED`） |
| POST | `/v1/payment/payments/callback/{channel}/{orderId}` | 渠道回调占位 |

## 冒烟

```bash
make smoke
# 或
BASE_URL=http://127.0.0.1:8180 ./scripts/smoke-payment-flow.sh
```
