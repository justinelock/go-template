package redislock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

var ErrNotAcquired = errors.New("lock not acquired")

// Lock 表示一次 Redis 分布式锁持有。
type Lock struct {
	// 步骤 1：Redis 客户端。
	client *redis.Client
	// 步骤 2：锁 key。
	key string
	// 步骤 3：持有者 token，Unlock 时校验。
	token string
}

// TryLock 尝试获取锁；成功返回 Lock，失败返回 ErrNotAcquired。
func TryLock(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (*Lock, error) {
	// 步骤 1：生成随机 token，用于释放时校验持有者。
	token, err := randomToken(16)
	if err != nil {
		return nil, err
	}

	// 步骤 2：SET key token NX EX ttl。
	ok, err := client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotAcquired
	}

	// 步骤 3：返回锁句柄。
	return &Lock{client: client, key: key, token: token}, nil
}

// Unlock 仅当 token 匹配时删除 key，避免误释放他人锁。
func (l *Lock) Unlock(ctx context.Context) error {
	// 步骤 1：Lua 脚本原子校验 token 并 DEL。
	const script = `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
	  return redis.call("DEL", KEYS[1])
	end
	return 0
	`
	_, err := l.client.Eval(ctx, script, []string{l.key}, l.token).Result()
	return err
}

func randomToken(n int) (string, error) {
	// 步骤 1：读取随机字节。
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	// 步骤 2：编码为 hex 字符串。
	return hex.EncodeToString(raw), nil
}
