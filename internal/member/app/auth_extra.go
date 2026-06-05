package app

import (
	"context"
	"strings"
	"time"

	"go-template/internal/member/domain"
	"go-template/internal/member/vo"
	"go-template/internal/platform/metrics"
)

// RefreshTokenTTL refresh token 在 Redis 中的存活时间。
const RefreshTokenTTL = 7 * 24 * time.Hour

// IntrospectResult token 反查结果，供网关 RBAC 与 gRPC/HTTP introspect 使用。
type IntrospectResult struct {
	// 步骤 1：用户 ID。
	UserID string
	// 步骤 2：角色（默认 user）。
	Role string
}

// Introspect 根据 access token 反查 userID 与 role。
func (s *Service) Introspect(ctx context.Context, token string) (*IntrospectResult, error) {
	// 步骤 1：Redis 反查 userID。
	userID, err := s.tokens.GetUserIDByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	// 步骤 2：加载用户记录取 role。
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	role := strings.TrimSpace(user.Role)
	if role == "" {
		role = "user"
	}
	return &IntrospectResult{UserID: userID, Role: role}, nil
}

// Refresh 使用 refresh token 换取新的 access/refresh token 对。
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*vo.LoginResp, error) {
	// 步骤 1：校验 refresh token 非空。
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, domain.ErrTokenRequired
	}
	// 步骤 2：反查 userID。
	userID, err := s.tokens.GetUserIDByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}
	// 步骤 3：加载用户资料。
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// 步骤 4：生成新 token 对。
	accessToken, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	newRefresh, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	// 步骤 5：写入 Redis 并轮换 refresh（删旧写新）。
	if err := s.tokens.SetToken(ctx, accessToken, userID, AccessTokenTTL); err != nil {
		return nil, err
	}
	_ = s.tokens.DeleteRefreshToken(ctx, refreshToken)
	if err := s.tokens.SetRefreshToken(ctx, newRefresh, userID, RefreshTokenTTL); err != nil {
		return nil, err
	}
	// 步骤 6：组装与登录一致的响应 VO。
	expiresInSec := int(AccessTokenTTL.Seconds())
	return &vo.LoginResp{
		Token: accessToken, Expire: expiresInSec, AccessToken: accessToken,
		RefreshToken: newRefresh, ID: user.ID, Username: user.Username,
		Email: user.Email, Mobile: user.Mobile, Nickname: user.Nickname,
		ParentID: user.ParentID, Level: user.Level, InviteCode: user.InviteCode,
		Status: user.Status, Verified: user.Verified, Avatar: user.Avatar,
		Remark: user.Remark, Role: user.Role, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}, nil
}

// maybeUpgradePassword 登录成功后将 MD5 密码懒升级为 bcrypt。
func (s *Service) maybeUpgradePassword(ctx context.Context, user *domain.UserRecord, plain string) {
	if !needsPasswordUpgrade(user.Password) {
		return
	}
	_ = s.users.UpdatePassword(ctx, user.ID, hashPassword(plain))
}

// recordLoginFailure 记录登录失败指标。
func (s *Service) recordLoginFailure() {
	metrics.AuthLoginFailTotal.Inc()
}
