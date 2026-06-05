package health

import (
	"net/http"

	"go-template/internal/platform/errcode"
	"go-template/internal/platform/httpx"
)

// LivenessHandler 进程存活探针；不探测外部依赖。
func LivenessHandler(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 步骤 1：透传 trace_id 并返回服务名。
		traceID := httpx.EnsureTraceID(r)
		httpx.JSON(w, http.StatusOK, traceID, errcode.OK, errcode.MsgOK, map[string]string{"service": serviceName})
	}
}

// ReadinessHandler 依赖就绪探针；prober 失败时返回 503。
func ReadinessHandler(serviceName string, prober *Prober) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 步骤 1：准备 trace_id。
		traceID := httpx.EnsureTraceID(r)
		// 步骤 2：执行聚合依赖检查。
		if prober != nil {
			if err := prober.Ready(r.Context()); err != nil {
				// 步骤 2.1：未就绪，返回 503 与原因。
				httpx.JSON(w, http.StatusServiceUnavailable, traceID, errcode.ServiceUnavailable, errcode.MsgServiceUnavailable, map[string]string{
					"service": serviceName,
					"reason":  err.Error(),
				})
				return
			}
		}
		// 步骤 3：全部通过，返回 ready。
		httpx.JSON(w, http.StatusOK, traceID, errcode.OK, errcode.MsgOK, map[string]string{"service": serviceName, "ready": "true"})
	}
}
