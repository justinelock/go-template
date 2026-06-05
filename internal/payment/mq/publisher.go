package paymentmq

import (
	"context"

	"go-template/internal/platform/mq"
)

// Publisher 向 payment.paid 等逻辑 Topic 发布消息。
type Publisher struct {
	// 步骤 1：统一消息总线。
	bus mq.Bus
}

// NewPublisher 构造支付域 MQ 发布器。
func NewPublisher(bus mq.Bus) *Publisher {
	return &Publisher{bus: bus}
}

// PublishPaid 支付成功后通知 order-service（逻辑 Topic payment.paid）。
func (p *Publisher) PublishPaid(ctx context.Context, orderID string) error {
	// 步骤 1：组装消息体（body 为 orderID）并注入 traceId。
	msg := mq.WithTraceHeader(ctx, mq.Message{
		Topic: mq.TopicPaymentPaid,
		Key:   orderID,
		Body:  []byte(orderID),
	})
	// 步骤 2：发布到 Bus。
	return p.bus.Publish(ctx, msg)
}
