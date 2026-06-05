package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	membergrpc "go-template/internal/gateway/client/membergrpc"
	"go-template/internal/gateway/routes"
)

// Resolver 将服务名解析为可达 HTTP 基址；失败时回退静态配置 URL。
type Resolver func(serviceName string, fallback string) string

// Authenticator 网关 token 校验器：
// 1) 优先 member gRPC introspect；
// 2) 失败回退 member HTTP introspect。
type Authenticator struct {
	// 步骤 1：调用 member HTTP introspect 的客户端。
	httpClient *http.Client
	// 步骤 2：member 静态回退基址。
	memberURL string
	// 步骤 3：Consul 等服务发现解析器。
	resolve Resolver
	// 步骤 4：可选 member gRPC 客户端。
	grpcClient *membergrpc.Client
}

// gatewayResp member HTTP 统一响应信封（仅解析 code/data）。
type gatewayResp struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

// introspectData introspect 接口 data 字段（兼容 userId / user_id）。
type introspectData struct {
	UserID       string `json:"userId"`
	UserIDLegacy string `json:"user_id"`
	Role         string `json:"role"`
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
func (a *Authenticator) Introspect(ctx context.Context, token string, traceID string) (AuthResult, error) {
	// 步骤 1：优先 gRPC introspect（短超时）。
	if a.grpcClient != nil {
		callCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
		defer cancel()
		userID, role, err := a.grpcClient.Introspect(callCtx, token, traceID)
		if err == nil {
			return AuthResult{UserID: userID, Role: role}, nil
		}
	}
	// 步骤 2：gRPC 失败则回退 HTTP introspect。
	return a.introspectByHTTP(token, traceID)
}

// introspectByHTTP 调用 member-service 的内部鉴权接口并解析 userID。
func (a *Authenticator) introspectByHTTP(token string, traceID string) (AuthResult, error) {
	// 步骤 1：解析 member 目标地址（优先服务发现，失败回退配置地址）。
	target := a.resolve("member-service", a.memberURL) + "/v1/auth/introspect"
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return AuthResult{}, err
	}

	// 步骤 2：透传 token 和 trace，确保下游可鉴权、可追踪。
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("token", token)
	req.Header.Set("X-Trace-Id", traceID)

	// 步骤 3：发起 HTTP 请求，网络错误直接返回。
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return AuthResult{}, err
	}
	defer resp.Body.Close()

	// 步骤 4：解析响应信封并校验业务 code。
	var parsed gatewayResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return AuthResult{}, err
	}
	if resp.StatusCode != http.StatusOK || parsed.Code != 0 {
		return AuthResult{}, fmt.Errorf("introspect rejected token")
	}

	// 步骤 5：解析 data 并规范化 userID/role。
	var data introspectData
	if err := json.Unmarshal(parsed.Data, &data); err != nil {
		return AuthResult{}, err
	}
	userID := strings.TrimSpace(data.UserID)
	if userID == "" {
		userID = strings.TrimSpace(data.UserIDLegacy)
	}
	if userID == "" {
		return AuthResult{}, fmt.Errorf("empty user id from introspect")
	}
	role := strings.TrimSpace(data.Role)
	if role == "" {
		role = "user"
	}
	return AuthResult{UserID: userID, Role: role}, nil
}

// RequiresAuth 根据网关路由表判断路径是否需鉴权。
func RequiresAuth(path string) bool {
	return routes.RequiresAuth(path)
}

// ExtractToken 委托 routes 包统一实现，便于测试与复用。
func ExtractToken(r *http.Request) string {
	return routes.ExtractToken(r)
}
