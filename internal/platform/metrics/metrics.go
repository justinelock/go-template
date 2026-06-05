// Package metrics 提供 Prometheus RED 指标与业务 counter。
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// httpRequestsTotal HTTP 请求计数（按 service/method/path/status）。
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests"},
		[]string{"service", "method", "path", "status"},
	)
	// httpRequestDuration HTTP 请求耗时直方图。
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "path"},
	)
	// OrderCreatedTotal 订单创建成功次数。
	OrderCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "order_created_total", Help: "Orders created"})
	// IdempotencyHitTotal 幂等缓存命中次数。
	IdempotencyHitTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "idempotency_hit_total", Help: "Idempotency cache hits"})
	// SettlementProcessedTotal 订单结算完成次数。
	SettlementProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "settlement_processed_total", Help: "Orders settled"})
	// AuthLoginFailTotal 登录失败次数。
	AuthLoginFailTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "auth_login_fail_total", Help: "Failed login attempts"})
	// PaymentCreatedTotal 支付单创建次数。
	PaymentCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "payment_created_total", Help: "Payments created"})
	// PaymentPaidTotal 支付完成次数。
	PaymentPaidTotal = promauto.NewCounter(prometheus.CounterOpts{Name: "payment_paid_total", Help: "Payments completed"})
)

// Handler 暴露 Prometheus 标准 /metrics 端点。
func Handler() http.Handler {
	return promhttp.Handler()
}

// statusWriter 捕获响应状态码供 RED 指标使用。
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware 记录 RED 指标（请求量 + 耗时）。
func HTTPMiddleware(service string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 步骤 1：记录开始时间与路径。
		start := time.Now()
		path := r.URL.Path
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		// 步骤 2：执行下游并捕获状态码。
		next.ServeHTTP(sw, r)
		// 步骤 3：写入 counter 与 histogram。
		status := strconv.Itoa(sw.status)
		httpRequestsTotal.WithLabelValues(service, r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(service, r.Method, path).Observe(time.Since(start).Seconds())
	})
}
