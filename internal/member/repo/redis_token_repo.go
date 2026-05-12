package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"pricing-assistant/internal/member/domain"
)

type RedisTokenRepo struct {
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
