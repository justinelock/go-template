package domain

import "errors"

const (
	// StatusPendingPayment 订单已创建，等待支付。
	StatusPendingPayment = "pending_payment"
	// StatusPaid 订单已支付，等待结算。
	StatusPaid = "paid"
	// StatusSettled 订单已结算完成。
	StatusSettled = "settled"
)

var (
	// ErrInvalidOrderInput 商品或金额等入参不合法。
	ErrInvalidOrderInput = errors.New("invalid order input")
	// ErrIdempotencyKeyRequired 缺少幂等键。
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	// ErrOrderNotFound 订单不存在或不属于当前用户。
	ErrOrderNotFound = errors.New("order not found")
	// ErrLockNotAcquired 并发创建时未获取分布式锁。
	ErrLockNotAcquired = errors.New("lock not acquired")
)

// CreateOrderReq 创建订单入参。
type CreateOrderReq struct {
	UserID         string
	ProductID      string
	Amount         float64
	IdempotencyKey string
}

// Order 订单实体。
type Order struct {
	ID             string
	UserID         string
	ProductID      string
	Amount         float64
	Status         string
	IdempotencyKey string
	CreatedAt      string
	UpdatedAt      string
}
