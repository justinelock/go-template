package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id"`
}

func EnsureTraceID(r *http.Request) string {
	traceID := r.Header.Get("X-Trace-Id")
	if traceID != "" {
		return traceID
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "trace_fallback"
	}
	return hex.EncodeToString(b)
}

func JSON(w http.ResponseWriter, status int, traceID string, code int, message string, data interface{}) {
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
