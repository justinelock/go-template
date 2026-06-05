package httptransport

import (
	"net"
	"net/http"
	"strings"

	"go-template/internal/gateway/routes"
	"go-template/internal/platform/errcode"
	"go-template/internal/platform/httpx"
	"go-template/internal/platform/ratelimit"
)

// withRateLimit 为网关入口按路由前缀 + 客户端 IP 限流。
func (h *Handler) withRateLimit(next http.Handler) http.Handler {
	// 步骤 1：未启用限流时直接透传。
	if h.rateLimiter == nil || !h.rateLimitEnabled {
		return next
	}
	rules := ratelimit.DefaultRules()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := httpx.EnsureTraceID(r)
		// 步骤 2：匹配第一条命中前缀的规则。
		for _, rule := range rules {
			if strings.HasPrefix(r.URL.Path, rule.Prefix) {
				ip := clientIP(r)
				key := rule.Prefix + ":" + ip
				ok, err := h.rateLimiter.Allow(r.Context(), key, rule.Limit, rule.Window)
				// 步骤 3：超限或 Redis 错误返回 429。
				if err != nil || !ok {
					httpx.JSON(w, http.StatusTooManyRequests, traceID, errcode.RateLimitExceeded, errcode.MsgRateLimitExceeded, nil)
					return
				}
				break
			}
		}
		// 步骤 4：放行到下游中间件。
		next.ServeHTTP(w, r)
	})
}

// clientIP 解析客户端 IP（优先 X-Forwarded-For）。
func clientIP(r *http.Request) string {
	// 步骤 1：代理场景取 X-Forwarded-For 首段。
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	// 步骤 2：回退 RemoteAddr。
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// roleAllowed 判断用户角色是否满足路由 RequiredRoles；空列表表示任意登录用户。
func roleAllowed(required []string, role string) bool {
	if len(required) == 0 {
		return true
	}
	for _, want := range required {
		if want == role {
			return true
		}
	}
	return false
}

// executeProxy 在断路器保护下执行下游 HTTP 调用。
func (h *Handler) executeProxy(serviceName string, fn func() error) error {
	// 步骤 1：未配置断路器池时直接执行。
	if h.breakerPool == nil {
		return fn()
	}
	// 步骤 2：按服务名获取断路器并执行。
	_, err := h.breakerPool.Get(serviceName).Execute(func() (interface{}, error) {
		return nil, fn()
	})
	return err
}

// checkRequiredRoles 校验路径所需角色；失败时已写 403 响应。
func checkRequiredRoles(path string, role string, w http.ResponseWriter, r *http.Request) bool {
	// 步骤 1：查询路径所需角色。
	required := routes.RequiredRolesForPath(path)
	// 步骤 2：不满足时写 403 并返回 false。
	if !roleAllowed(required, role) {
		traceID := httpx.EnsureTraceID(r)
		httpx.JSON(w, http.StatusForbidden, traceID, errcode.Forbidden, errcode.MsgForbidden, nil)
		return false
	}
	return true
}
