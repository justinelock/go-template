package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Response 全服务统一 JSON 响应信封。
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id"`
}

// EnsureTraceID 从 W3C traceparent、X-Trace-Id、OTel context 或随机值生成 trace_id。
func EnsureTraceID(r *http.Request) string {
	// 步骤 1：优先 W3C traceparent 第二段。
	if parent := strings.TrimSpace(r.Header.Get("traceparent")); parent != "" {
		if parts := strings.Split(parent, "-"); len(parts) >= 2 && parts[1] != "" {
			return parts[1]
		}
	}
	// 步骤 2：兼容 X-Trace-Id 请求头。
	if traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id")); traceID != "" {
		return traceID
	}
	// 步骤 3：从 OTel span context 读取。
	if sc := trace.SpanFromContext(r.Context()).SpanContext(); sc.HasTraceID() {
		return sc.TraceID().String()
	}
	// 步骤 4：生成随机 fallback trace_id。
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "trace_fallback"
	}
	return hex.EncodeToString(b)
}

// JSON 写入统一响应信封并设置 X-Trace-Id。
func JSON(w http.ResponseWriter, status int, traceID string, code int, message string, data interface{}) {
	// 步骤 1：设置 Content-Type 与 trace 响应头。
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Trace-Id", traceID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Data:    data,
		TraceID: traceID,
	})
}
