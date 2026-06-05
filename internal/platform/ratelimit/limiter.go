// Package ratelimit 提供基于 Redis 的网关固定窗口限流。
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Limiter 基于 Redis INCR+EXPIRE 的固定窗口限流器。
type Limiter struct {
	// 步骤 1：Redis 客户端。
	redis *redis.Client
	// 步骤 2：键前缀，避免与其他业务冲突。
	prefix string
}

// New 构造限流器；prefix 为空时使用 "ratelimit"。
func New(rdb *redis.Client, prefix string) *Limiter {
	if prefix == "" {
		prefix = "ratelimit"
	}
	return &Limiter{redis: rdb, prefix: prefix}
}

// Allow 在 window 内允许最多 limit 次；超限返回 false。
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// 步骤 1：未启用或非法参数时直接放行。
	if l == nil || l.redis == nil || limit <= 0 {
		return true, nil
	}
	// 步骤 2：Redis 事务 INCR 并设置过期。
	redisKey := fmt.Sprintf("%s:%s", l.prefix, key)
	pipe := l.redis.TxPipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	// 步骤 3：比较计数与限额。
	return incr.Val() <= int64(limit), nil
}

// RouteRule 单条路由限流规则（前缀 + 限额 + 窗口）。
type RouteRule struct {
	Prefix string
	Limit  int
	Window time.Duration
}

// DefaultRules 网关默认限流规则（登录/注册/下单）。
func DefaultRules() []RouteRule {
	return []RouteRule{
		{Prefix: "/v1/auth/login", Limit: 10, Window: time.Minute},
		{Prefix: "/v1/auth/register", Limit: 5, Window: time.Minute},
		{Prefix: "/v1/order/orders", Limit: 60, Window: time.Minute},
	}
}
