package routes

import (
	"net/http"
	"strings"
)

// ExtractToken 统一 token 提取优先级：Authorization Bearer > token header > token query。
func ExtractToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	if token := strings.TrimSpace(r.Header.Get("token")); token != "" {
		return token
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}
