package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrNotFound = errors.New("idempotency record not found")

// Record 幂等键对应的响应快照。
type Record struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
}

// Store Redis 幂等存储：idempotency:{scope}:{key} -> JSON。
type Store struct {
	// 步骤 1：Redis 客户端。
	redis *redis.Client
	// 步骤 2：键 TTL，过期后允许新业务语义（Demo 默认 24h）。
	ttl time.Duration
}

// NewStore 创建幂等存储，默认 TTL 24h。
func NewStore(client *redis.Client, ttl time.Duration) *Store {
	// 步骤 1：TTL 非法时使用默认值。
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	// 步骤 2：返回 Store 实例。
	return &Store{redis: client, ttl: ttl}
}

// key 生成 Redis 键名：idempotency:{scope}:{idempotencyKey}。
func (s *Store) key(scope, idempotencyKey string) string {
	return "idempotency:" + scope + ":" + idempotencyKey
}

// Get 读取已存在的幂等结果。
func (s *Store) Get(ctx context.Context, scope, idempotencyKey string) (*Record, error) {
	// 步骤 1：从 Redis 读取 JSON。
	raw, err := s.redis.Get(ctx, s.key(scope, idempotencyKey)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// 步骤 2：反序列化为 Record。
	var rec Record
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// Set 写入幂等结果。
func (s *Store) Set(ctx context.Context, scope, idempotencyKey string, rec Record) error {
	// 步骤 1：序列化并 SET EX。
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, s.key(scope, idempotencyKey), raw, s.ttl).Err()
}
