package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-template/internal/member/domain"

	"github.com/go-redis/redis/v8"
)

// RedisTokenRepo 基于 Redis 的 access/refresh token 存储实现。
type RedisTokenRepo struct {
	// 步骤 1：Redis 客户端（token:{access}、refresh:{refresh} 键空间）。
	redis *redis.Client
}

func NewRedisTokenRepo(redisClient *redis.Client) *RedisTokenRepo {
	// 步骤 1：注入 Redis 客户端并返回 token 仓储实例。
	return &RedisTokenRepo{redis: redisClient}
}

// SetToken 写入 token -> userID 映射，并设置 TTL。
func (r *RedisTokenRepo) SetToken(ctx context.Context, token, userID string, ttl time.Duration) error {
	// 步骤 1：标准化 token 后落库，键格式为 token:<token>。
	return r.redis.Set(ctx, "token:"+strings.TrimSpace(token), userID, ttl).Err()
}

// DeleteToken 删除 token 映射（登出时使用）。
func (r *RedisTokenRepo) DeleteToken(ctx context.Context, token string) error {
	// 步骤 1：标准化 token 并删除对应 key。
	return r.redis.Del(ctx, "token:"+strings.TrimSpace(token)).Err()
}

// GetUserIDByToken 通过 token 反查 userID。
func (r *RedisTokenRepo) GetUserIDByToken(ctx context.Context, token string) (string, error) {
	// 步骤 1：校验 token 必填。
	if strings.TrimSpace(token) == "" {
		return "", domain.ErrTokenRequired
	}

	// 步骤 2：查询 Redis 中 token 映射。
	userID, err := r.redis.Get(ctx, "token:"+strings.TrimSpace(token)).Result()
	if errors.Is(err, redis.Nil) {
		// 步骤 3：未命中映射为 token 无效错误。
		return "", domain.ErrTokenInvalid
	}
	if err != nil {
		// 步骤 4：Redis 访问异常透传。
		return "", err
	}

	// 步骤 5：返回命中的 userID。
	return userID, nil
}

// SetRefreshToken 写入 refresh:{token} -> userID 映射。
func (r *RedisTokenRepo) SetRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error {
	// 步骤 1：标准化 token 后写入 Redis 并设置 TTL。
	return r.redis.Set(ctx, "refresh:"+strings.TrimSpace(token), userID, ttl).Err()
}

// GetUserIDByRefreshToken 通过 refresh token 反查 userID。
func (r *RedisTokenRepo) GetUserIDByRefreshToken(ctx context.Context, token string) (string, error) {
	// 步骤 1：校验 token 必填。
	if strings.TrimSpace(token) == "" {
		return "", domain.ErrTokenRequired
	}
	// 步骤 2：查询 refresh 映射。
	userID, err := r.redis.Get(ctx, "refresh:"+strings.TrimSpace(token)).Result()
	if errors.Is(err, redis.Nil) {
		return "", domain.ErrTokenInvalid
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

// DeleteRefreshToken 删除 refresh token（登出或轮换时调用）。
func (r *RedisTokenRepo) DeleteRefreshToken(ctx context.Context, token string) error {
	// 步骤 1：删除 refresh 键。
	return r.redis.Del(ctx, "refresh:"+strings.TrimSpace(token)).Err()
}
