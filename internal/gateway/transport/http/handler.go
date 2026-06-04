package httptransport

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	gatewayapp "go-template/internal/gateway/app"
	"go-template/internal/gateway/vo"
	"go-template/internal/platform/httpx"
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
	// 步骤 5：跨域配置。
	corsAllowOrigin string
}

// NewHandler 组装网关 HTTP 处理器依赖。
func NewHandler(
	httpClient *http.Client,
	resolve gatewayapp.Resolver,
	authenticator *gatewayapp.Authenticator,
	memberURL string,
	corsAllowOrigin string,
) *Handler {
	// 步骤 1：注入依赖并返回可复用 Handler 实例。
	return &Handler{
		httpClient:      httpClient,
		resolve:         resolve,
		authenticator:   authenticator,
		memberURL:       memberURL,
		corsAllowOrigin: corsAllowOrigin,
	}
}

// BuildServer 按顺序装配网关中间层：
// 先鉴权，再 CORS（保证错误响应也带跨域头）。
func (h *Handler) BuildServer(mux *http.ServeMux) http.Handler {
	// 步骤 1：链式包裹中间件并返回最终入口 handler。
	return h.withCORS(h.withAuth(mux))
}

// RegisterRoutes 注册网关公开路由。
// 约束：网关仅做鉴权、路由、协议转发，不承载业务实现。
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 步骤 1：基础健康检查与公开配置。
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/v1/public/config", h.publicConfig)

	// 步骤 2：认证相关接口（转发 member-service）。
	mux.HandleFunc("/v1/auth/login", h.proxyToMember("/v1/auth/login"))
	mux.HandleFunc("/v1/auth/register", h.proxyToMember("/v1/auth/register"))
	mux.HandleFunc("/v1/auth/logout", h.proxyToMember("/v1/auth/logout"))

	// 步骤 3：member 域用户接口（新前缀 -> member 内部路径）。
	mux.HandleFunc("/v1/member/users/profile", h.proxyToMember("/v1/users/profile"))

	// 步骤 4：WebSocket 占位（规范见 docs/api/websocket.md）。
	mux.HandleFunc("/v1/ws", h.websocketPlaceholder)
}

// websocketPlaceholder WebSocket 未实现时返回 501 与业务码 50101。
func (h *Handler) websocketPlaceholder(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：生成/透传 traceID，与 REST 接口保持一致。
	traceID := httpx.EnsureTraceID(r)
	// 步骤 2：返回 501 与业务码 50101，告知客户端能力尚未开放（见 docs/api/websocket.md）。
	httpx.JSON(w, http.StatusNotImplemented, traceID, 50101, "websocket not implemented", nil)
}

// healthz 返回网关健康状态。
func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：生成/透传 traceID，便于诊断链路。
	traceID := httpx.EnsureTraceID(r)

	// 步骤 2：输出统一响应结构。
	httpx.JSON(w, http.StatusOK, traceID, 0, "ok", vo.HealthResp{Service: "gateway-service"})
}

// publicConfig 返回前端启动所需的公开配置。
func (h *Handler) publicConfig(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：准备 traceID 并返回静态配置。
	traceID := httpx.EnsureTraceID(r)
	httpx.JSON(w, http.StatusOK, traceID, 0, "ok", vo.PublicConfigResp{
		Env: "local",
	})
}

// proxyToMember 构造固定路径代理（member-service）。
func (h *Handler) proxyToMember(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 步骤 1：解析目标地址。
		target := h.resolve("member-service", h.memberURL) + path

		// 步骤 2：透传 query 参数，避免筛选条件丢失。
		target = appendRawQuery(target, r)

		// 步骤 3：执行统一代理。
		h.proxy(w, r, target)
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
func (h *Handler) proxy(w http.ResponseWriter, r *http.Request, target string) {
	// 步骤 1：准备 traceID 并读取原始请求体。
	traceID := httpx.EnsureTraceID(r)
	body, _ := ioReadAll(r.Body)

	// 步骤 2：基于原请求方法构造下游请求。
	req, err := http.NewRequest(r.Method, target, bytes.NewReader(body))
	if err != nil {
		httpx.JSON(w, http.StatusBadGateway, traceID, 50001, "proxy build request failed", nil)
		return
	}

	// 步骤 3：复制关键请求头，保持鉴权与链路信息完整。
	copyHeaders(req, r, traceID)

	// 步骤 4：调用下游服务。
	resp, err := h.httpClient.Do(req)
	if err != nil {
		httpx.JSON(w, http.StatusBadGateway, traceID, 50002, "downstream service unavailable", nil)
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
			httpx.JSON(w, http.StatusUnauthorized, traceID, 40101, "token is required", nil)
			return
		}

		// 步骤 3：调用鉴权器校验 token 并获取 userID。
		userID, err := h.authenticator.Introspect(r.Context(), token, traceID)
		if err != nil {
			httpx.JSON(w, http.StatusUnauthorized, traceID, 40102, "token is invalid or expired", nil)
			return
		}

		// 步骤 4：克隆请求并注入用户上下文，继续后续处理链。
		cloned := r.Clone(r.Context())
		cloned.Header.Set("X-User-Id", userID)
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
