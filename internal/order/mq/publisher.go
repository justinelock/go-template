package ordermq

import (
	"context"

	orderapp "go-template/internal/order/app"
	"go-template/internal/platform/mq"
)

// SettlePublisher 将平台 Bus 适配为 order app 的 Publisher 接口。
type SettlePublisher struct {
	// 步骤 1：统一消息总线，底层可为 RabbitMQ 或 RocketMQ。
	bus mq.Bus
}

// NewSettlePublisher 构造结算消息发布器。
func NewSettlePublisher(bus mq.Bus) *SettlePublisher {
	return &SettlePublisher{bus: bus}
}

// PublishSettle 向逻辑 Topic order.settle 发布订单 ID。
func (p *SettlePublisher) PublishSettle(ctx context.Context, orderID string) error {
	// 步骤 1：组装平台 Message 并发布。
	return p.bus.Publish(ctx, mq.Message{
		Topic: mq.TopicOrderSettle,
		Key:   orderID,
		Body:  []byte(orderID),
	})
}

var _ orderapp.Publisher = (*SettlePublisher)(nil)
