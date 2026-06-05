package grpctransport

import (
	"context"
	"errors"
	"strings"

	memberv1 "go-template/api/gen/member/v1"
	memberapp "go-template/internal/member/app"
	"go-template/internal/member/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServer 实现 member AuthService gRPC，供网关 token introspect。
type AuthServer struct {
	memberv1.UnimplementedAuthServiceServer
	// 步骤 1：member 领域服务（Redis token + MySQL user）。
	svc *memberapp.Service
}

// NewAuthServer 注入 member app 服务。
func NewAuthServer(svc *memberapp.Service) *AuthServer {
	return &AuthServer{svc: svc}
}

// Introspect 根据 access token 返回 userID 与 role。
func (s *AuthServer) Introspect(ctx context.Context, req *memberv1.IntrospectRequest) (*memberv1.IntrospectResponse, error) {
	// 步骤 1：校验 token 必填。
	token := strings.TrimSpace(req.GetToken())
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	// 步骤 2：调用 app 层反查 userID/role。
	result, err := s.svc.Introspect(ctx, token)
	// 步骤 3：领域错误映射 gRPC 状态码。
	if errors.Is(err, domain.ErrTokenInvalid) || errors.Is(err, domain.ErrTokenRequired) {
		return nil, status.Error(codes.Unauthenticated, "token is invalid or expired")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "token verify failed")
	}
	// 步骤 4：组装成功响应。
	return &memberv1.IntrospectResponse{
		Code:    0,
		Message: "ok",
		UserId:  result.UserID,
		Role:    result.Role,
		TraceId: req.GetTraceId(),
	}, nil
}
