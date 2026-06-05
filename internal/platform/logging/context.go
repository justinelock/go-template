// Package logging 提供 slog 初始化、HTTP 访问日志与 trace_id 上下文传递。
package logging

import "context"

// ctxKey 用于在 context 中存储 trace_id，避免与第三方键冲突。
type ctxKey int

const traceIDKey ctxKey = 1

// WithTraceID 将 trace_id 写入 context，供 app/worker 日志关联。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	// 步骤 1：以私有 key 写入 trace_id。
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext 从 context 读取 trace_id；未设置时返回空串。
func TraceIDFromContext(ctx context.Context) string {
	// 步骤 1：类型断言读取，失败则视为未设置。
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}
