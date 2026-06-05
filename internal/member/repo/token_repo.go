package repo

import (
	"context"
	"time"
)

// TokenRepository 定义 member 域 token 存储能力：
// - 用于 access token 的写入、删除、反查 userID；
// - 典型实现为 Redis，接口保持存储无关性。
type TokenRepository interface {
	// SetToken 保存 token -> userID 映射并设置过期时间。
	SetToken(ctx context.Context, token, userID string, ttl time.Duration) error
	// DeleteToken 删除 token 映射（登出场景）。
	DeleteToken(ctx context.Context, token string) error
	// GetUserIDByToken 通过 token 反查 userID（鉴权场景）。
	GetUserIDByToken(ctx context.Context, token string) (string, error)
	// SetRefreshToken 保存 refresh token。
	SetRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error
	// GetUserIDByRefreshToken 反查 refresh token。
	GetUserIDByRefreshToken(ctx context.Context, token string) (string, error)
	// DeleteRefreshToken 删除 refresh token。
	DeleteRefreshToken(ctx context.Context, token string) error
}
