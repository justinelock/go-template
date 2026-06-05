package mq

import (
	"context"

	"go-template/internal/platform/logging"
)

// headerTraceID MQ 消息头中承载的链路 ID 键名（与 REST X-Trace-Id 对应）。
const headerTraceID = "traceId"

// WithTraceHeader 从 context 注入 traceId 到消息头，供 worker 日志关联。
func WithTraceHeader(ctx context.Context, msg Message) Message {
	// 步骤 1：确保 Headers map 已初始化。
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	// 步骤 2：从 logging context 读取 trace_id 并写入。
	if tid := logging.TraceIDFromContext(ctx); tid != "" {
		msg.Headers[headerTraceID] = tid
	}
	return msg
}

// TraceIDFromMessage 从消息头读取 traceId；无则返回空串。
func TraceIDFromMessage(msg Message) string {
	if msg.Headers == nil {
		return ""
	}
	return msg.Headers[headerTraceID]
}
