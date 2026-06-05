// Package health 提供 liveness/readiness HTTP 处理器与依赖探测聚合。
package health

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-redis/redis/v8"
)

// Check 单项就绪探测函数；返回 nil 表示该项健康。
type Check func(ctx context.Context) error

// Prober 聚合多项依赖探测（MySQL、Redis、MQ 等）。
type Prober struct {
	// 步骤 1：按注册顺序依次执行的检查列表。
	checks []Check
}

// NewProber 构造就绪探测器，传入零个或多个 Check。
func NewProber(checks ...Check) *Prober {
	// 步骤 1：保存检查函数切片。
	return &Prober{checks: checks}
}

// Ready 全部检查通过返回 nil；任一失败返回首个错误。
func (p *Prober) Ready(ctx context.Context) error {
	// 步骤 1：顺序执行各项检查。
	for _, check := range p.checks {
		if check == nil {
			continue
		}
		if err := check(ctx); err != nil {
			return err
		}
	}
	return nil
}

// MySQL 返回 MySQL Ping 检查函数（2s 超时）。
func MySQL(db *sql.DB) Check {
	return func(ctx context.Context) error {
		// 步骤 1：带超时 Ping 数据库连接池。
		c, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return db.PingContext(c)
	}
}

// Redis 返回 Redis Ping 检查函数（2s 超时）。
func Redis(rdb *redis.Client) Check {
	return func(ctx context.Context) error {
		// 步骤 1：带超时 Ping Redis。
		c, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return rdb.Ping(c).Err()
	}
}

// PingFunc 包装任意 Ping 实现（如 mq.Bus.Ping）。
type PingFunc func(ctx context.Context) error

// FromPing 将 Ping 转为 Check；fn 为 nil 时返回 nil。
func FromPing(fn PingFunc) Check {
	if fn == nil {
		return nil
	}
	return func(ctx context.Context) error {
		// 步骤 1：带超时调用自定义 Ping。
		c, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return fn(c)
	}
}
