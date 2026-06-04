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

type AuthServer struct {
	memberv1.UnimplementedAuthServiceServer
	svc *memberapp.Service
}

func NewAuthServer(svc *memberapp.Service) *AuthServer {
	return &AuthServer{svc: svc}
}

func (s *AuthServer) Introspect(ctx context.Context, req *memberv1.IntrospectRequest) (*memberv1.IntrospectResponse, error) {
	token := strings.TrimSpace(req.GetToken())
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}
	userID, err := s.svc.IntrospectToken(ctx, token)
	if errors.Is(err, domain.ErrTokenInvalid) || errors.Is(err, domain.ErrTokenRequired) {
		return nil, status.Error(codes.Unauthenticated, "token is invalid or expired")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "token verify failed")
	}
	return &memberv1.IntrospectResponse{
		Code:    0,
		Message: "ok",
		UserId:  userID,
		TraceId: req.GetTraceId(),
	}, nil
}
