// Package httpserver 装配各服务通用的 HTTP 中间件链。
package httpserver

import (
	"net/http"

	"go-template/internal/platform/httpx"
	"go-template/internal/platform/logging"
	"go-template/internal/platform/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Chain 按顺序装配通用 HTTP 中间件链：访问日志 → recovery → 指标 →（可选）OTel。
func Chain(service string, mux http.Handler, otelEnabled bool) http.Handler {
	// 步骤 1：从业务 mux 出发，自内向外包裹。
	h := http.Handler(mux)
	// 步骤 2：访问日志（含 trace_id 注入 context）。
	h = logging.AccessMiddleware(service, h)
	// 步骤 3：panic 恢复，避免进程崩溃。
	h = httpx.RecoveryMiddleware(h)
	// 步骤 4：Prometheus RED 指标。
	h = metrics.HTTPMiddleware(service, h)
	// 步骤 5：可选 OpenTelemetry HTTP 包装。
	if otelEnabled {
		h = otelhttp.NewHandler(h, service)
	}
	return h
}
