package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-template/internal/order/domain"
	"go-template/internal/platform/timefmt"
)

// MySQLOrderRepo 基于 MySQL orders 表的订单仓储实现。
type MySQLOrderRepo struct {
	// 步骤 1：数据库连接池。
	db *sql.DB
}

// NewMySQLOrderRepo 构造 MySQL 订单仓储。
func NewMySQLOrderRepo(db *sql.DB) *MySQLOrderRepo {
	// 步骤 1：注入 DB 并返回仓储实例。
	return &MySQLOrderRepo{db: db}
}

// Create 插入 pending 订单并返回自增 ID。
func (r *MySQLOrderRepo) Create(ctx context.Context, order domain.Order) (string, error) {
	// 步骤 1：插入 pending 订单。
	now := time.Now()
	res, err := r.db.ExecContext(ctx, `
INSERT INTO orders (user_id, product_id, amount, status, idempotency_key, created_at)
VALUES (?, ?, ?, 0, ?, ?)`,
		order.UserID, order.ProductID, order.Amount, order.IdempotencyKey, now)
	if err != nil {
		return "", err
	}
	// 步骤 2：读取自增主键并转为字符串。
	id, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", id), nil
}

// GetByID 按用户 + 订单 ID 查询（防止越权）。
func (r *MySQLOrderRepo) GetByID(ctx context.Context, userID, orderID string) (*domain.Order, error) {
	// 步骤 1：按 id + user_id 查询单行。
	row := r.db.QueryRowContext(ctx, `
SELECT id, user_id, product_id, amount, status, idempotency_key, created_at, updated_at
FROM orders WHERE id = ? AND user_id = ?`, orderID, userID)
	// 步骤 2：扫描并映射领域实体。
	return scanOrder(row)
}

// GetByIdempotencyKey 按用户 + 幂等键查询已有订单。
func (r *MySQLOrderRepo) GetByIdempotencyKey(ctx context.Context, userID, key string) (*domain.Order, error) {
	// 步骤 1：按唯一键 uk_user_idempotency 查询。
	row := r.db.QueryRowContext(ctx, `
SELECT id, user_id, product_id, amount, status, idempotency_key, created_at, updated_at
FROM orders WHERE user_id = ? AND idempotency_key = ?`, userID, key)
	order, err := scanOrder(row)
	if errors.Is(err, domain.ErrOrderNotFound) {
		return nil, domain.ErrOrderNotFound
	}
	return order, err
}

// MarkSettled 将 pending 订单更新为 settled。
func (r *MySQLOrderRepo) MarkSettled(ctx context.Context, orderID string) error {
	// 步骤 1：条件更新 status=1，仅 pending 可被结算。
	now := time.Now()
	res, err := r.db.ExecContext(ctx, `
UPDATE orders SET status = 1, updated_at = ? WHERE id = ? AND status = 0`, now, orderID)
	if err != nil {
		return err
	}
	// 步骤 2：忽略已结算或不存在（RowsAffected=0 视为幂等成功）。
	_, err = res.RowsAffected()
	return err
}

// scanOrder 将 SQL 行扫描为 domain.Order。
func scanOrder(row *sql.Row) (*domain.Order, error) {
	var (
		id             int64
		userID         int64
		productID      string
		amount         float64
		status         int
		idempotencyKey string
		createdAt      time.Time
		updatedAt      sql.NullTime
	)
	// 步骤 1：扫描列值。
	if err := row.Scan(&id, &userID, &productID, &amount, &status, &idempotencyKey, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}
	// 步骤 2：映射 status 整型为领域字符串。
	orderStatus := domain.StatusPending
	if status == 1 {
		orderStatus = domain.StatusSettled
	}
	// 步骤 3：组装 Order 实体。
	order := &domain.Order{
		ID:             fmt.Sprintf("%d", id),
		UserID:         fmt.Sprintf("%d", userID),
		ProductID:      productID,
		Amount:         amount,
		Status:         orderStatus,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      timefmt.DateTimeSecond(createdAt),
	}
	if updatedAt.Valid {
		order.UpdatedAt = timefmt.DateTimeSecond(updatedAt.Time)
	}
	return order, nil
}
