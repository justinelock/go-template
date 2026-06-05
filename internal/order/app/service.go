package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-template/internal/order/domain"
	"go-template/internal/order/repo"
	"go-template/internal/order/vo"
	"go-template/internal/platform/idempotency"
	"go-template/internal/platform/mq"
	"go-template/internal/platform/redislock"

	"github.com/go-redis/redis/v8"
)

const lockTTL = 30 * time.Second

// Publisher 发布结算消息的抽象，便于测试替换。
type Publisher interface {
	PublishSettle(ctx context.Context, orderID string) error
}

// Service 订单用例：
// 1) 幂等创建订单并投递 MQ；
// 2) 查询订单状态；
// 3) 异步结算更新状态。
type Service struct {
	// 步骤 1：订单仓储，负责 MySQL 读写。
	orders repo.OrderRepository
	// 步骤 2：Redis 幂等快照，加速重复请求响应。
	idempotency *idempotency.Store
	// 步骤 3：Redis 客户端，供分布式锁使用。
	redis *redis.Client
	// 步骤 4：MQ 发布器，投递结算消息。
	publisher Publisher
}

// NewService 通过依赖注入组装订单用例服务。
func NewService(orders repo.OrderRepository, idem *idempotency.Store, redisClient *redis.Client, publisher Publisher) *Service {
	// 步骤 1：注入依赖并返回 service 实例。
	return &Service{
		orders:      orders,
		idempotency: idem,
		redis:       redisClient,
		publisher:   publisher,
	}
}

// CreateOrder 同步落库并投递结算消息；相同幂等键返回同一订单。
func (s *Service) CreateOrder(ctx context.Context, req domain.CreateOrderReq) (*vo.CreateOrderResp, error) {
	// 步骤 1：基础校验。
	req.UserID = strings.TrimSpace(req.UserID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if req.UserID == "" || req.ProductID == "" || req.Amount <= 0 {
		return nil, domain.ErrInvalidOrderInput
	}
	if req.IdempotencyKey == "" {
		return nil, domain.ErrIdempotencyKeyRequired
	}

	scope := "order:" + req.UserID

	// 步骤 2：Redis 幂等快路径。
	if rec, err := s.idempotency.Get(ctx, scope, req.IdempotencyKey); err == nil {
		return &vo.CreateOrderResp{OrderID: rec.OrderID, Status: rec.Status}, nil
	} else if !errors.Is(err, idempotency.ErrNotFound) {
		return nil, err
	}

	// 步骤 3：DB 幂等兜底（Redis 未命中时）。
	if existing, err := s.orders.GetByIdempotencyKey(ctx, req.UserID, req.IdempotencyKey); err == nil {
		resp := &vo.CreateOrderResp{OrderID: existing.ID, Status: existing.Status}
		_ = s.idempotency.Set(ctx, scope, req.IdempotencyKey, idempotency.Record{OrderID: existing.ID, Status: existing.Status})
		return resp, nil
	} else if !errors.Is(err, domain.ErrOrderNotFound) {
		return nil, err
	}

	// 步骤 4：分布式锁，避免并发重复创建。
	lockKey := fmt.Sprintf("lock:order:create:%s:%s", req.UserID, req.IdempotencyKey)
	lock, err := redislock.TryLock(ctx, s.redis, lockKey, lockTTL)
	if err != nil {
		if errors.Is(err, redislock.ErrNotAcquired) {
			return nil, domain.ErrLockNotAcquired
		}
		return nil, err
	}
	defer func() { _ = lock.Unlock(ctx) }()

	// 步骤 5：持锁后再查幂等（双检）。
	if rec, err := s.idempotency.Get(ctx, scope, req.IdempotencyKey); err == nil {
		return &vo.CreateOrderResp{OrderID: rec.OrderID, Status: rec.Status}, nil
	}
	if existing, err := s.orders.GetByIdempotencyKey(ctx, req.UserID, req.IdempotencyKey); err == nil {
		resp := &vo.CreateOrderResp{OrderID: existing.ID, Status: existing.Status}
		_ = s.idempotency.Set(ctx, scope, req.IdempotencyKey, idempotency.Record{OrderID: existing.ID, Status: existing.Status})
		return resp, nil
	}

	// 步骤 6：写入 pending 订单。
	orderID, err := s.orders.Create(ctx, domain.Order{
		UserID:         req.UserID,
		ProductID:      req.ProductID,
		Amount:         req.Amount,
		Status:         domain.StatusPending,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	// 步骤 7：发布结算消息。
	if err := s.publisher.PublishSettle(ctx, orderID); err != nil {
		return nil, err
	}

	// 步骤 8：缓存幂等结果。
	resp := &vo.CreateOrderResp{OrderID: orderID, Status: domain.StatusPending}
	_ = s.idempotency.Set(ctx, scope, req.IdempotencyKey, idempotency.Record{OrderID: orderID, Status: domain.StatusPending})
	return resp, nil
}

// GetOrder 按 ID 查询当前用户订单并映射为 VO。
func (s *Service) GetOrder(ctx context.Context, userID, orderID string) (*vo.OrderResp, error) {
	// 步骤 1：查询订单实体。
	order, err := s.orders.GetByID(ctx, strings.TrimSpace(userID), strings.TrimSpace(orderID))
	if err != nil {
		return nil, err
	}
	// 步骤 2：映射为对外 VO。
	out := vo.FromDomain(order.ID, order.UserID, order.ProductID, order.Amount, order.Status, order.IdempotencyKey, order.CreatedAt, order.UpdatedAt)
	return &out, nil
}

// SettleOrder 将订单标记为 settled（worker 调用）。
func (s *Service) SettleOrder(ctx context.Context, orderID string) error {
	// 步骤 1：更新 MySQL 状态为 settled。
	return s.orders.MarkSettled(ctx, strings.TrimSpace(orderID))
}

// EnsurePublisher 类型断言 helper，供 main 使用。
var _ Publisher = (*mq.Client)(nil)
