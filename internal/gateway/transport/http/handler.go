package httptransport

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	gatewayapp "go-template/internal/gateway/app"
	"go-template/internal/gateway/routes"
	"go-template/internal/gateway/vo"
	"go-template/internal/platform/errcode"
	"go-template/internal/platform/gatewayresilience"
	"go-template/internal/platform/httpx"
	"go-template/internal/platform/ratelimit"
)

// Handler 负责 API Gateway 的 HTTP 入站处理：
// 1) 注册对外 API 路由；
// 2) 执行统一鉴权/CORS 中间层；
// 3) 将请求代理到对应下游服务。
type Handler struct {
	// 步骤 1：基础 HTTP 客户端，用于转发请求到下游服务。
	httpClient *http.Client
	// 步骤 2：服务发现解析器（优先 Consul，失败回退静态地址）。
	resolve gatewayapp.Resolver
	// 步骤 3：网关鉴权器，负责 token -> userID 的统一校验。
	authenticator *gatewayapp.Authenticator
	// 步骤 4：member-service 回退地址（当服务发现失败时使用）。
	memberURL string
	// 步骤 5：order-service 回退地址。
	orderURL string
	// 步骤 5b：payment-service 回退地址。
	paymentURL string
	// 步骤 6：跨域配置。
	corsAllowOrigin string
	// 步骤 7：限流与断路器（可选）。
	rateLimiter      *ratelimit.Limiter
	rateLimitEnabled bool
	breakerPool      *gatewayresilience.BreakerPool
}

// NewHandler 组装网关 HTTP 处理器依赖。
func NewHandler(
	httpClient *http.Client,
	resolve gatewayapp.Resolver,
	authenticator *gatewayapp.Authenticator,
	memberURL string,
	orderURL string,
	paymentURL string,
	corsAllowOrigin string,
	rateLimiter *ratelimit.Limiter,
	rateLimitEnabled bool,
	breakerPool *gatewayresilience.BreakerPool,
) *Handler {
	// 步骤 1：注入依赖并返回可复用 Handler 实例。
	return &Handler{
		httpClient:       httpClient,
		resolve:          resolve,
		authenticator:    authenticator,
		memberURL:        memberURL,
		orderURL:         orderURL,
		paymentURL:       paymentURL,
		corsAllowOrigin:  corsAllowOrigin,
		rateLimiter:      rateLimiter,
		rateLimitEnabled: rateLimitEnabled,
		breakerPool:      breakerPool,
	}
}

// BuildServer 按顺序装配网关中间层：
// 先鉴权，再 CORS（保证错误响应也带跨域头）。
func (h *Handler) BuildServer(mux *http.ServeMux) http.Handler {
	// 步骤 1：链式包裹中间件并返回最终入口 handler。
	return h.withCORS(h.withRateLimit(h.withAuth(mux)))
}

// RegisterRoutes 注册网关公开路由。
// 约束：网关仅做鉴权、路由、协议转发，不承载业务实现。
//
// 业务代理路由不再逐条注册到 mux（标准库 mux 无法运行时增删），
// 改为统一挂载到 catch-all "/"，请求时查 routes 原子快照分发，从而支持热加载。
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 步骤 1：公开配置（显式路由优先级高于 catch-all）。
	mux.HandleFunc("/v1/public/config", h.publicConfig)

	// 步骤 2：WebSocket 占位（规范见 docs/api/websocket.md）。
	mux.HandleFunc("/v1/ws", h.websocketPlaceholder)

	// 步骤 3：catch-all 动态分发，按可热加载的路由表代理到下游。
	mux.HandleFunc("/", h.dispatch)
}

// dispatch 按运行时路由快照（支持热加载）将请求代理到下游服务。
func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：查路由表（原子快照，精确优先、前缀兜底）。
	route, ok := routes.Match(r.URL.Path)
	if !ok {
		traceID := httpx.EnsureTraceID(r)
		httpx.JSON(w, http.StatusNotFound, traceID, errcode.RouteNotFound, errcode.MsgRouteNotFound, nil)
		return
	}

	// 步骤 2：解析下游基址（优先路由自带直连基址，其次服务发现/静态回退）。
	fallback := route.UpstreamBaseURL
	if fallback == "" {
		fallback = h.serviceFallbackURL(route.ServiceName)
	}
	base := h.resolve(route.ServiceName, fallback)

	// 步骤 3：拼接下游完整 URL（前缀路由追加尾段，如 /{id}）。
	target := base + route.UpstreamTargetPath(r.URL.Path)

	// 步骤 4：透传 query 参数后执行反向代理。
	target = appendRawQuery(target, r)
	h.proxy(w, r, route.ServiceName, target)
}

// websocketPlaceholder WebSocket 未实现时返回 501 与业务码 50101。
func (h *Handler) websocketPlaceholder(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：生成/透传 traceID，与 REST 接口保持一致。
	traceID := httpx.EnsureTraceID(r)
	// 步骤 2：返回 501 与业务码 50101，告知客户端能力尚未开放（见 docs/api/websocket.md）。
	httpx.JSON(w, http.StatusNotImplemented, traceID, errcode.WebSocketNotImplemented, errcode.MsgWebSocketNotImplemented, nil)
}

// publicConfig 返回前端启动所需的公开配置。
func (h *Handler) publicConfig(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：准备 traceID 并返回静态配置。
	traceID := httpx.EnsureTraceID(r)
	httpx.JSON(w, http.StatusOK, traceID, errcode.OK, errcode.MsgOK, vo.PublicConfigResp{
		Env: "local",
	})
}

// serviceFallbackURL 返回服务发现失败时的静态回退基址。
func (h *Handler) serviceFallbackURL(serviceName string) string {
	switch serviceName {
	case "member-service":
		return h.memberURL
	case "order-service":
		return h.orderURL
	case "payment-service":
		return h.paymentURL
	default:
		return ""
	}
}

// appendRawQuery 将请求 query 原样拼接到目标 URL。
func appendRawQuery(target string, r *http.Request) string {
	if raw := r.URL.RawQuery; raw != "" {
		if strings.Contains(target, "?") {
			return target + "&" + raw
		}
		return target + "?" + raw
	}
	return target
}

// proxy 执行网关统一反向代理流程：
// - 复制请求体
// - 构造下游请求
// - 透传关键头
// - 回写下游响应
func (h *Handler) proxy(w http.ResponseWriter, r *http.Request, serviceName, target string) {
	// 步骤 1：准备 traceID 并读取原始请求体。
	traceID := httpx.EnsureTraceID(r)
	body, _ := ioReadAll(r.Body)

	// 步骤 2：基于原请求方法构造下游请求。
	req, err := http.NewRequest(r.Method, target, bytes.NewReader(body))
	if err != nil {
		httpx.JSON(w, http.StatusBadGateway, traceID, errcode.ProxyBuildFailed, errcode.MsgProxyBuildFailed, nil)
		return
	}

	// 步骤 3：复制关键请求头，保持鉴权与链路信息完整。
	copyHeaders(req, r, traceID)

	// 步骤 4：调用下游服务（可选断路器）。
	var resp *http.Response
	err = h.executeProxy(serviceName, func() error {
		var doErr error
		resp, doErr = h.httpClient.Do(req)
		return doErr
	})
	if err != nil {
		httpx.JSON(w, http.StatusBadGateway, traceID, errcode.DownstreamUnavailable, errcode.MsgDownstreamUnavailable, nil)
		return
	}
	defer resp.Body.Close()
	// 步骤 5：读取并原样回写下游响应体。
	data, _ := ioReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Trace-Id", traceID)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(data)
}

// copyHeaders 复制网关到下游所需的关键请求头。
func copyHeaders(req *http.Request, src *http.Request, traceID string) {
	// 步骤 1：保留内容类型与鉴权头。
	req.Header.Set("Content-Type", src.Header.Get("Content-Type"))
	req.Header.Set("Authorization", src.Header.Get("Authorization"))
	req.Header.Set("token", src.Header.Get("token"))

	// 步骤 2：透传幂等键与用户上下文。
	req.Header.Set("X-Idempotency-Key", src.Header.Get("X-Idempotency-Key"))
	req.Header.Set("X-User-Id", src.Header.Get("X-User-Id"))
	req.Header.Set("X-User-Role", src.Header.Get("X-User-Role"))

	// 步骤 3：统一写入网关 traceID，保证链路可追踪。
	req.Header.Set("X-Trace-Id", traceID)
}

// withAuth 为需要鉴权的路由统一执行 token 校验，并注入 X-User-Id。
func (h *Handler) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 步骤 1：无需鉴权的路径直接放行。
		if !gatewayapp.RequiresAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// 步骤 2：提取 token，缺失直接返回 401。
		traceID := httpx.EnsureTraceID(r)
		token := gatewayapp.ExtractToken(r)
		if token == "" {
			httpx.JSON(w, http.StatusUnauthorized, traceID, errcode.TokenRequired, errcode.MsgTokenRequired, nil)
			return
		}

		// 步骤 3：调用鉴权器校验 token 并获取 userID。
		auth, err := h.authenticator.Introspect(r.Context(), token, traceID)
		if err != nil {
			httpx.JSON(w, http.StatusUnauthorized, traceID, errcode.TokenInvalid, errcode.MsgTokenInvalid, nil)
			return
		}
		if !checkRequiredRoles(r.URL.Path, auth.Role, w, r) {
			return
		}

		cloned := r.Clone(r.Context())
		cloned.Header.Set("X-User-Id", auth.UserID)
		cloned.Header.Set("X-User-Role", auth.Role)
		next.ServeHTTP(w, cloned)
	})
}

// withCORS 为所有响应统一写入跨域头，含预检请求处理。
func (h *Handler) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 步骤 1：先写入统一 CORS 响应头。
		w.Header().Set("Access-Control-Allow-Origin", h.corsAllowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,token,Content-Type,X-Trace-Id,X-Idempotency-Key")
		w.Header().Set("Access-Control-Expose-Headers", "X-Trace-Id")

		// 步骤 2：预检请求直接返回 204。
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 步骤 3：非预检请求继续下发。
		next.ServeHTTP(w, r)
	})
}

// ioReadAll 是 io.ReadAll 的小包装：
// 1) 方便单元测试时做替换；
// 2) 避免在调用处引入额外辅助依赖。
func ioReadAll(r io.Reader) ([]byte, error) {
	// 步骤 1：读取完整流并返回。
	return io.ReadAll(r)
}
