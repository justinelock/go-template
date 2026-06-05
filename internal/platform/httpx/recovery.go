package httpx

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// RecoveryMiddleware 捕获 handler panic 并返回统一 500 JSON。
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 步骤 1：defer recover，防止 panic 导致进程退出。
		defer func() {
			if rec := recover(); rec != nil {
				// 步骤 2：记录 trace_id、路径、堆栈。
				traceID := EnsureTraceID(r)
				slog.Error("panic recovered",
					"trace_id", traceID,
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				// 步骤 3：返回统一错误信封。
				JSON(w, http.StatusInternalServerError, traceID, 50000, "internal server error", nil)
			}
		}()
		// 步骤 4：正常执行下游。
		next.ServeHTTP(w, r)
	})
}
