package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"go-template/internal/payment/domain"
	"go-template/internal/payment/repo"
	"go-template/internal/platform/metrics"
)

// PaidPublisher 支付成功后的 MQ 发布抽象，便于测试替换。
type PaidPublisher interface {
	PublishPaid(ctx context.Context, orderID string) error
}

// Service 封装 payment 领域用例：
// 1) 消费 payment.created 创建支付单；
// 2) mock-pay / 渠道回调标记已支付；
// 3) 发布 payment.paid 驱动订单状态机。
type Service struct {
	// 步骤 1：支付单仓储。
	payments repo.PaymentRepository
	// 步骤 2：支付成功事件发布器。
	publisher PaidPublisher
	// 步骤 3：是否允许 dev mock-pay。
	mockEnabled bool
}

// NewService 通过依赖注入组装 payment 用例服务。
func NewService(payments repo.PaymentRepository, publisher PaidPublisher, mockEnabled bool) *Service {
	return &Service{payments: payments, publisher: publisher, mockEnabled: mockEnabled}
}

// paymentCreatedEvent MQ payment.created 消息体结构。
type paymentCreatedEvent struct {
	OrderID string  `json:"order_id"`
	UserID  string  `json:"user_id"`
	Amount  float64 `json:"amount"`
}

// HandlePaymentCreated 处理 order-service 发出的 payment.created 消息。
func (s *Service) HandlePaymentCreated(ctx context.Context, body []byte) error {
	// 步骤 1：反序列化消息体。
	var evt paymentCreatedEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return err
	}
	// 步骤 2：幂等创建支付单（按 order_id 唯一）。
	_, err := s.EnsurePayment(ctx, domain.CreatePaymentReq{
		OrderID: strings.TrimSpace(evt.OrderID),
		UserID:  strings.TrimSpace(evt.UserID),
		Amount:  evt.Amount,
		Channel: "internal",
	})
	return err
}

// EnsurePayment 按 order_id 幂等创建 pending 支付单。
func (s *Service) EnsurePayment(ctx context.Context, req domain.CreatePaymentReq) (string, error) {
	// 步骤 1：清洗并校验入参。
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.UserID = strings.TrimSpace(req.UserID)
	if req.OrderID == "" || req.UserID == "" || req.Amount <= 0 {
		return "", domain.ErrInvalidInput
	}
	// 步骤 2：已存在则直接返回（幂等）。
	if existing, err := s.payments.GetByOrderID(ctx, req.OrderID); err == nil {
		return existing.ID, nil
	} else if !errors.Is(err, domain.ErrPaymentNotFound) {
		return "", err
	}
	// 步骤 3：插入 pending 记录。
	id, err := s.payments.Create(ctx, domain.Payment{
		OrderID: req.OrderID, UserID: req.UserID, Amount: req.Amount,
		Status: domain.StatusPending, Channel: req.Channel,
	})
	if err != nil {
		return "", err
	}
	// 步骤 4：业务指标 +1。
	metrics.PaymentCreatedTotal.Inc()
	return id, nil
}

// MockPay dev 环境模拟支付成功并发布 payment.paid。
func (s *Service) MockPay(ctx context.Context, paymentID string) error {
	// 步骤 1：校验 mock 开关。
	if !s.mockEnabled {
		return domain.ErrMockPayDisabled
	}
	// 步骤 2：加载支付单并校验状态。
	payment, err := s.payments.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment.Status == domain.StatusPaid {
		return domain.ErrAlreadyPaid
	}
	// 步骤 3：更新为 paid。
	if err := s.payments.MarkPaid(ctx, paymentID); err != nil {
		return err
	}
	// 步骤 4：指标 +1 并通知 order-service。
	metrics.PaymentPaidTotal.Inc()
	return s.publisher.PublishPaid(ctx, payment.OrderID)
}

// GetPayment 按 ID 查询支付单。
func (s *Service) GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	return s.payments.GetByID(ctx, strings.TrimSpace(paymentID))
}

// Callback 渠道回调占位：按 orderID 标记已支付（验签留待接入真实渠道）。
func (s *Service) Callback(ctx context.Context, channel, orderID string) error {
	// 步骤 1：按订单查支付单。
	payment, err := s.payments.GetByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	// 步骤 2：已支付则幂等成功。
	if payment.Status == domain.StatusPaid {
		return nil
	}
	// 步骤 3：标记 paid 并发布 MQ（channel 仅记录用途，当前未落库更新）。
	if err := s.payments.MarkPaid(ctx, payment.ID); err != nil {
		return err
	}
	metrics.PaymentPaidTotal.Inc()
	return s.publisher.PublishPaid(ctx, payment.OrderID)
}
