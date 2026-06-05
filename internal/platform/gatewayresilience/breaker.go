// Package gatewayresilience 提供网关下游调用的断路器池。
package gatewayresilience

import (
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

// BreakerPool 按下游服务名维护独立断路器实例。
type BreakerPool struct {
	// 步骤 1：保护 breakers map 的互斥锁。
	mu sync.Mutex
	// 步骤 2：serviceName -> CircuitBreaker。
	breakers map[string]*gobreaker.CircuitBreaker
}

// NewBreakerPool 构造空断路器池。
func NewBreakerPool() *BreakerPool {
	return &BreakerPool{breakers: make(map[string]*gobreaker.CircuitBreaker)}
}

// Get 获取或懒创建指定服务的断路器。
func (p *BreakerPool) Get(serviceName string) *gobreaker.CircuitBreaker {
	// 步骤 1：读缓存，命中则返回。
	p.mu.Lock()
	defer p.mu.Unlock()
	if cb, ok := p.breakers[serviceName]; ok {
		return cb
	}
	// 步骤 2：按 SMB 默认策略创建断路器（连续 5 次失败开路 30s）。
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        serviceName,
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})
	p.breakers[serviceName] = cb
	return cb
}
