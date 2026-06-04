package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	membergrpc "go-template/internal/gateway/client/membergrpc"
)

type Resolver func(serviceName string, fallback string) string

type Authenticator struct {
	httpClient *http.Client
	memberURL  string
	resolve    Resolver
	grpcClient *membergrpc.Client
}

type gatewayResp struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

type introspectData struct {
	UserID       string `json:"userId"`
	UserIDLegacy string `json:"user_id"`
}

// NewAuthenticator 组装网关鉴权器，支持 gRPC 优先、HTTP 回退。
func NewAuthenticator(httpClient *http.Client, memberURL string, resolve Resolver, grpcClient *membergrpc.Client) *Authenticator {
	return &Authenticator{
		httpClient: httpClient,
		memberURL:  memberURL,
		resolve:    resolve,
		grpcClient: grpcClient,
	}
}

// Introspect 统一 token 校验入口：
// 1) 优先走 member gRPC；
// 2) gRPC 不可用/失败时回退 HTTP introspect。
func (a *Authenticator) Introspect(ctx context.Context, token string, traceID string) (string, error) {
	if a.grpcClient != nil {
		// 1) gRPC 调用设置短超时，避免网关鉴权链路被长时间阻塞。
		callCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
		defer cancel()
		userID, err := a.grpcClient.Introspect(callCtx, token, traceID)
		if err == nil {
			return userID, nil
		}
		// gRPC 失败后自动回退 HTTP，不直接中断请求。
	}
	// 2) 回退到 HTTP introspect（兼容兜底路径）。
	return a.introspectByHTTP(token, traceID)
}

// introspectByHTTP 调用 member-service 的内部鉴权接口并解析 userID。
func (a *Authenticator) introspectByHTTP(token string, traceID string) (string, error) {
	// 1) 解析 member 目标地址（优先服务发现，失败回退配置地址）。
	target := a.resolve("member-service", a.memberURL) + "/v1/auth/introspect"
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}

	// 2) 透传 token 和 trace，确保下游可鉴权、可追踪。
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("token", token)
	req.Header.Set("X-Trace-Id", traceID)

	// 3) 发起 HTTP 请求，网络错误直接返回。
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 4) 解析统一响应壳并校验业务成功码。
	var parsed gatewayResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK || parsed.Code != 0 {
		return "", fmt.Errorf("introspect rejected token")
	}

	// 5) 解析 data.user_id 并做非空校验。
	var data introspectData
	if err := json.Unmarshal(parsed.Data, &data); err != nil {
		return "", err
	}
	userID := strings.TrimSpace(data.UserID)
	if userID == "" {
		userID = strings.TrimSpace(data.UserIDLegacy)
	}
	if userID == "" {
		return "", fmt.Errorf("empty user id from introspect")
	}
	return userID, nil
}

// RequiresAuth 标记需要网关鉴权的路径白名单（保护账户/订单/用户私有接口）。
func RequiresAuth(path string) bool {
	if path == "/v1/auth/logout" {
		return true
	}
	if path == "/v1/member/users/profile" {
		return true
	}
	return false
}

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
