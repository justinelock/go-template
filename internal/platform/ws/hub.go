package ws

import "net/http"

// Hub 管理 WebSocket 连接生命周期（占位接口，完整实现待后续迭代）。
type Hub interface {
	// ServeHTTP 处理 WebSocket 升级或占位响应。
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NoopHub 未实现时的占位：由调用方决定 HTTP 响应（如 501）。
type NoopHub struct{}

func (NoopHub) ServeHTTP(http.ResponseWriter, *http.Request) {}
