package membergrpc

import (
	"context"
	"fmt"
	"strings"

	memberv1 "go-template/api/gen/member/v1"
)

type Client struct {
	enabled bool
	addr    string
	client  memberv1.AuthServiceClient
}

func New(enabled bool, addr string, client memberv1.AuthServiceClient) *Client {
	return &Client{
		enabled: enabled,
		addr:    addr,
		client:  client,
	}
}

func (c *Client) Introspect(ctx context.Context, token string, traceID string) (string, error) {
	if !c.enabled || c.client == nil {
		return "", fmt.Errorf("member grpc client disabled")
	}
	resp, err := c.client.Introspect(ctx, &memberv1.IntrospectRequest{
		Token:   token,
		TraceId: traceID,
	})
	if err != nil {
		return "", err
	}

	if resp.GetCode() != 0 {
		return "", fmt.Errorf("grpc introspect rejected token, code=%d", resp.GetCode())
	}

	userID := strings.TrimSpace(resp.GetUserId())
	if userID == "" {
		return "", fmt.Errorf("empty user id from grpc introspect")
	}
	return userID, nil
}

func (c *Client) Addr() string {
	return c.addr
}
