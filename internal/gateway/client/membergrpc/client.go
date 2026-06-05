package membergrpc

import (
	"context"
	"fmt"
	"strings"

	memberv1 "go-template/api/gen/member/v1"
)

// Client 网关侧 member AuthService gRPC 客户端封装。
type Client struct {
	// 步骤 1：是否启用 gRPC introspect。
	enabled bool
	// 步骤 2：gRPC 地址（日志/诊断用）。
	addr string
	// 步骤 3：底层 AuthServiceClient。
	client memberv1.AuthServiceClient
}

// New 构造 member gRPC 客户端；enabled=false 时 Introspect 直接失败以便 HTTP 回退。
func New(enabled bool, addr string, client memberv1.AuthServiceClient) *Client {
	return &Client{
		enabled: enabled,
		addr:    addr,
		client:  client,
	}
}

// Introspect 调用 member gRPC 校验 token，返回 userID 与 role。
func (c *Client) Introspect(ctx context.Context, token string, traceID string) (string, string, error) {
	// 步骤 1：未启用时返回错误，触发网关 HTTP 回退。
	if !c.enabled || c.client == nil {
		return "", "", fmt.Errorf("member grpc client disabled")
	}
	// 步骤 2：发起 gRPC introspect 并校验业务 code。
	resp, err := c.client.Introspect(ctx, &memberv1.IntrospectRequest{
		Token:   token,
		TraceId: traceID,
	})
	if err != nil {
		return "", "", err
	}

	if resp.GetCode() != 0 {
		return "", "", fmt.Errorf("grpc introspect rejected token, code=%d", resp.GetCode())
	}

	// 步骤 3：规范化 userID 与 role（空 role 默认 user）。
	userID := strings.TrimSpace(resp.GetUserId())
	if userID == "" {
		return "", "", fmt.Errorf("empty user id from grpc introspect")
	}
	role := strings.TrimSpace(resp.GetRole())
	if role == "" {
		role = "user"
	}
	return userID, role, nil
}

// Addr 返回配置的 gRPC 地址。
func (c *Client) Addr() string {
	return c.addr
}
