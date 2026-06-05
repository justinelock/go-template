package ordermq

import (
	"context"
	"encoding/json"

	orderapp "go-template/internal/order/app"
	"go-template/internal/platform/mq"
)

// EventPublisher 将平台 Bus 适配为 order app 的 Publisher 接口。
type EventPublisher struct {
	// 步骤 1：统一消息总线。
	bus mq.Bus
}

// NewEventPublisher 构造订单事件发布器。
func NewEventPublisher(bus mq.Bus) *EventPublisher {
	return &EventPublisher{bus: bus}
}

// paymentCreatedPayload order → payment 的 payment.created 消息体。
type paymentCreatedPayload struct {
	OrderID string  `json:"order_id"`
	UserID  string  `json:"user_id"`
	Amount  float64 `json:"amount"`
}

// PublishPaymentCreated 通知 payment-service 创建支付单。
func (p *EventPublisher) PublishPaymentCreated(ctx context.Context, orderID, userID string, amount float64) error {
	// 步骤 1：序列化订单上下文供 payment 消费。
	body, _ := json.Marshal(paymentCreatedPayload{OrderID: orderID, UserID: userID, Amount: amount})
	msg := mq.WithTraceHeader(ctx, mq.Message{
		Topic: mq.TopicPaymentCreated,
		Key:   orderID,
		Body:  body,
	})
	return p.bus.Publish(ctx, msg)
}

// PublishSettle 向逻辑 Topic order.settle 发布订单 ID。
func (p *EventPublisher) PublishSettle(ctx context.Context, orderID string) error {
	// 步骤 1：投递结算消息（body 为 orderID 字符串）。
	msg := mq.WithTraceHeader(ctx, mq.Message{
		Topic: mq.TopicOrderSettle,
		Key:   orderID,
		Body:  []byte(orderID),
	})
	return p.bus.Publish(ctx, msg)
}

var _ orderapp.Publisher = (*EventPublisher)(nil)

// NewSettlePublisher 兼容旧名。
func NewSettlePublisher(bus mq.Bus) *EventPublisher {
	return NewEventPublisher(bus)
}
