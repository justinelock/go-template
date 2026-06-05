package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-template/internal/payment/domain"
)

// MySQLPaymentRepo 基于 MySQL payments 表的仓储实现。
type MySQLPaymentRepo struct {
	// 步骤 1：数据库连接池。
	db *sql.DB
}

// NewMySQLPaymentRepo 构造 MySQL 支付仓储。
func NewMySQLPaymentRepo(db *sql.DB) *MySQLPaymentRepo {
	return &MySQLPaymentRepo{db: db}
}

// Create 插入 status=0(pending) 支付单并返回自增 ID。
func (r *MySQLPaymentRepo) Create(ctx context.Context, payment domain.Payment) (string, error) {
	// 步骤 1：写入 payments 表。
	now := time.Now()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO payments (order_id, user_id, amount, status, channel, created_at)
VALUES (?, ?, ?, 0, ?, ?)`,
		payment.OrderID, payment.UserID, payment.Amount, payment.Channel, now)
	if err != nil {
		return "", err
	}
	// 步骤 2：读取自增主键。
	id, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", id), nil
}

// GetByID 按支付单主键查询。
func (r *MySQLPaymentRepo) GetByID(ctx context.Context, paymentID string) (*domain.Payment, error) {
	row := r.db.QueryRowContext(ctx, selectPaymentSQL("id = ?"), paymentID)
	return scanPayment(row)
}

// GetByOrderID 按订单 ID 查询（uk_order_id）。
func (r *MySQLPaymentRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	row := r.db.QueryRowContext(ctx, selectPaymentSQL("order_id = ?"), orderID)
	return scanPayment(row)
}

// MarkPaid 条件更新 status=1，仅 pending 可变为 paid。
func (r *MySQLPaymentRepo) MarkPaid(ctx context.Context, paymentID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
UPDATE payments SET status = 1, updated_at = ? WHERE id = ? AND status = 0`, now, paymentID)
	return err
}

// selectPaymentSQL 生成带 WHERE 子句的查询 SQL。
func selectPaymentSQL(where string) string {
	return fmt.Sprintf(`
SELECT id, order_id, user_id, amount, status, COALESCE(channel,''),
       DATE_FORMAT(created_at, '%%Y-%%m-%%d %%H:%%i:%%s'),
       COALESCE(DATE_FORMAT(updated_at, '%%Y-%%m-%%d %%H:%%i:%%s'), '')
FROM payments WHERE %s LIMIT 1`, where)
}

// scanPayment 将 SQL 行映射为 domain.Payment。
func scanPayment(row *sql.Row) (*domain.Payment, error) {
	var (
		id, orderID, userID, channel string
		amount                       float64
		status                       int
		createdAt, updatedAt         string
	)
	// 步骤 1：扫描列。
	if err := row.Scan(&id, &orderID, &userID, &amount, &status, &channel, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, err
	}
	// 步骤 2：映射 status 整型为领域字符串。
	st := domain.StatusPending
	if status == 1 {
		st = domain.StatusPaid
	}
	return &domain.Payment{
		ID: id, OrderID: orderID, UserID: userID, Amount: amount,
		Status: st, Channel: channel, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

// ToVO 将领域实体转为 HTTP 响应 map（驼峰字段）。
func ToVO(p *domain.Payment) map[string]any {
	return map[string]any{
		"id": p.ID, "orderId": p.OrderID, "userId": p.UserID,
		"amount": p.Amount, "status": p.Status, "channel": p.Channel,
		"createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt,
	}
}
