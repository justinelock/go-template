package worker

import (
	"context"
	"log"
	"time"

	orderapp "go-template/internal/order/app"
	"go-template/internal/platform/mq"
)

// StartSettlement 启动 RabbitMQ 结算消费者（同进程 Demo）。
func StartSettlement(mqClient *mq.Client, svc *orderapp.Service) error {
	// 步骤 1：注册 consume handler。
	return mqClient.ConsumeSettle(func(orderID string) error {
		// 步骤 2：模拟结算耗时。
		time.Sleep(200 * time.Millisecond)

		// 步骤 3：更新订单状态为 settled。
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := svc.SettleOrder(ctx, orderID); err != nil {
			log.Printf("settlement failed order=%s err=%v", orderID, err)
			return err
		}
		log.Printf("order settled: %s", orderID)
		return nil
	})
}
