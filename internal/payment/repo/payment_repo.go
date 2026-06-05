package repo

import (
	"context"

	"go-template/internal/payment/domain"
)

// PaymentRepository 支付单持久化接口；app 层仅依赖此抽象。
type PaymentRepository interface {
	// Create 插入 pending 支付单并返回 ID。
	Create(ctx context.Context, payment domain.Payment) (string, error)
	// GetByID 按主键查询。
	GetByID(ctx context.Context, paymentID string) (*domain.Payment, error)
	// GetByOrderID 按订单 ID 查询（uk_order_id 唯一）。
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	// MarkPaid 将 pending 更新为 paid。
	MarkPaid(ctx context.Context, paymentID string) error
}
