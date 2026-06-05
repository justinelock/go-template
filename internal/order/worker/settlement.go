package worker

import (
	"context"
	"log"
	"time"

	orderapp "go-template/internal/order/app"
	"go-template/internal/platform/mq"
)

// StartSettlement 订阅 order.settle 并异步更新订单为 settled。
func StartSettlement(bus mq.Bus, svc *orderapp.Service) error {
	// 步骤 1：后台长期运行的消费上下文（进程退出时随 main 结束）。
	ctx := context.Background()

	// 步骤 2：订阅逻辑 Topic + ConsumerGroup，与 MQ 实现无关。
	return bus.Subscribe(ctx, mq.TopicOrderSettle, mq.GroupOrderSettlement, func(ctx context.Context, msg mq.Message) error {
		orderID := string(msg.Body)

		// 步骤 3：模拟结算耗时。
		time.Sleep(200 * time.Millisecond)

		// 步骤 4：更新订单状态为 settled。
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := svc.SettleOrder(callCtx, orderID); err != nil {
			log.Printf("settlement failed order=%s err=%v", orderID, err)
			return err
		}
		log.Printf("order settled: %s", orderID)
		return nil
	})
}
