package worker

import (
	"context"
	"log/slog"

	paymentapp "go-template/internal/payment/app"
	"go-template/internal/platform/logging"
	"go-template/internal/platform/mq"
)

// StartPaymentCreated 订阅 payment.created，为订单创建支付单。
func StartPaymentCreated(bus mq.Bus, svc *paymentapp.Service) error {
	// 步骤 1：后台长期消费上下文。
	ctx := context.Background()
	// 步骤 2：订阅逻辑 Topic + ConsumerGroup。
	return bus.Subscribe(ctx, mq.TopicPaymentCreated, mq.GroupPaymentCreator, func(ctx context.Context, msg mq.Message) error {
		// 步骤 3：从 MQ 头恢复 trace_id 到 context。
		if tid := mq.TraceIDFromMessage(msg); tid != "" {
			ctx = logging.WithTraceID(ctx, tid)
		}
		// 步骤 4：解析消息并幂等创建支付单。
		if err := svc.HandlePaymentCreated(ctx, msg.Body); err != nil {
			slog.Error("payment created handler failed", "err", err)
			return err
		}
		return nil
	})
}
