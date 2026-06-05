// Package domain 定义 payment-service 领域实体、状态常量与错误。
package domain

import "errors"

const (
	// StatusPending 支付单已创建，等待支付。
	StatusPending = "pending"
	// StatusPaid 支付已完成。
	StatusPaid = "paid"
	// StatusFailed 支付失败（预留）。
	StatusFailed = "failed"
)

var (
	// ErrInvalidInput 入参不合法（订单/用户/金额等）。
	ErrInvalidInput = errors.New("invalid payment input")
	// ErrPaymentNotFound 支付单不存在。
	ErrPaymentNotFound = errors.New("payment not found")
	// ErrMockPayDisabled 生产环境关闭 mock-pay。
	ErrMockPayDisabled = errors.New("mock pay disabled")
	// ErrAlreadyPaid 重复支付。
	ErrAlreadyPaid = errors.New("payment already paid")
)

// Payment 支付单实体（与 payments 表对应）。
type Payment struct {
	ID        string
	OrderID   string
	UserID    string
	Amount    float64
	Status    string
	Channel   string
	CreatedAt string
	UpdatedAt string
}

// CreatePaymentReq 创建支付单入参。
type CreatePaymentReq struct {
	OrderID string
	UserID  string
	Amount  float64
	Channel string
}
