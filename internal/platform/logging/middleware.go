package logging

import (
	"log/slog"
	"net/http"
	"time"

	"go-template/internal/platform/httpx"
)

// statusWriter 包装 ResponseWriter 以捕获实际 HTTP 状态码。
type statusWriter struct {
	http.ResponseWriter
	// 步骤 1：默认 200，WriteHeader 时更新。
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	// 步骤 1：记录状态码并透传。
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// AccessMiddleware 记录 HTTP 访问日志（method/path/status/latency/trace_id/user_id）。
func AccessMiddleware(service string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 步骤 1：记录开始时间并确保 trace_id 存在。
		start := time.Now()
		traceID := httpx.EnsureTraceID(r)
		// 步骤 2：将 trace_id 注入 request context，供下游 handler 使用。
		ctx := WithTraceID(r.Context(), traceID)
		r = r.WithContext(ctx)

		// 步骤 3：包装 writer 并执行下游链。
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		// 步骤 4：组装结构化访问日志字段。
		attrs := []any{
			"service", service,
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"latency_ms", time.Since(start).Milliseconds(),
			"trace_id", traceID,
		}
		if uid := r.Header.Get("X-User-Id"); uid != "" {
			attrs = append(attrs, "user_id", uid)
		}
		// 步骤 5：输出访问日志。
		slog.Info("http_access", attrs...)
	})
}
