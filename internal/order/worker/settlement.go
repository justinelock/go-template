package worker

import (
	"context"
	"log/slog"
	"time"

	orderapp "go-template/internal/order/app"
	"go-template/internal/platform/logging"
	"go-template/internal/platform/mq"
)

// StartSettlement 订阅 order.settle 并异步更新订单为 settled。
func StartSettlement(bus mq.Bus, svc *orderapp.Service) error {
	// 步骤 1：后台长期运行的消费上下文（进程退出时随 main 结束）。
	ctx := context.Background()

	// 步骤 2：订阅逻辑 Topic + ConsumerGroup，与 MQ 实现无关。
	return bus.Subscribe(ctx, mq.TopicOrderSettle, mq.GroupOrderSettlement, func(ctx context.Context, msg mq.Message) error {
		if tid := mq.TraceIDFromMessage(msg); tid != "" {
			ctx = logging.WithTraceID(ctx, tid)
		}
		orderID := string(msg.Body)
		time.Sleep(200 * time.Millisecond)
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := svc.SettleOrder(callCtx, orderID); err != nil {
			slog.Error("settlement failed", "order_id", orderID, "err", err)
			return err
		}
		slog.Info("order settled", "order_id", orderID)
		return nil
	})
}
