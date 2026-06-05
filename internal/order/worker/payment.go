package worker

import (
	"context"
	"log/slog"

	orderapp "go-template/internal/order/app"
	"go-template/internal/platform/logging"
	"go-template/internal/platform/mq"
)

// StartPaymentPaid 订阅 payment.paid，将订单标记为 paid 并投递 order.settle。
func StartPaymentPaid(bus mq.Bus, svc *orderapp.Service) error {
	// 步骤 1：后台长期消费上下文。
	ctx := context.Background()
	// 步骤 2：订阅 payment.paid + order 消费组。
	return bus.Subscribe(ctx, mq.TopicPaymentPaid, mq.GroupOrderPaymentPaid, func(ctx context.Context, msg mq.Message) error {
		// 步骤 3：恢复 trace_id。
		if tid := mq.TraceIDFromMessage(msg); tid != "" {
			ctx = logging.WithTraceID(ctx, tid)
		}
		orderID := string(msg.Body)
		// 步骤 4：更新订单为 paid 并发布结算消息。
		if err := svc.MarkPaidAndSettle(ctx, orderID); err != nil {
			slog.Error("payment paid handler failed", "order_id", orderID, "err", err)
			return err
		}
		slog.Info("order paid and settle queued", "order_id", orderID)
		return nil
	})
}
