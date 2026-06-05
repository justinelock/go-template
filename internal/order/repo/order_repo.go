package repo

import (
	"context"

	"go-template/internal/order/domain"
)

// OrderRepository 订单持久化接口。
type OrderRepository interface {
	Create(ctx context.Context, order domain.Order) (string, error)
	GetByID(ctx context.Context, userID, orderID string) (*domain.Order, error)
	GetByIdempotencyKey(ctx context.Context, userID, key string) (*domain.Order, error)
	MarkSettled(ctx context.Context, orderID string) error
}
