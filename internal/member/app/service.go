package app

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"go-template/internal/member/domain"
	"go-template/internal/member/repo"
	"go-template/internal/member/vo"
)

const AccessTokenTTL = 12 * time.Hour

// Service 封装 member 领域核心用例：
// - 用户鉴权（登录/登出/token 校验）
// - 用户资料查询与更新
type Service struct {
	// 步骤 1：用户仓储，负责用户资料与签到记录读写。
	users repo.UserRepository
	// 步骤 2：token 仓储，负责 token 存储与反查。
	tokens repo.TokenRepository
}

// NewService 通过依赖注入组装用例服务，便于测试替换 repo/client。
func NewService(users repo.UserRepository, tokens repo.TokenRepository) *Service {
	// 步骤 1：注入依赖并返回 service 实例。
	return &Service{
		users:  users,
		tokens: tokens,
	}
}

// Login 支持 username/mobile 双入口登录，并返回兼容新老前端字段的 VO。
func (s *Service) Login(ctx context.Context, req domain.LoginReq) (*vo.LoginResp, error) {
	// 步骤 1：兼容 username/mobile 双入口；优先 username，缺失回退 mobile。
	account := strings.TrimSpace(req.Username)
	if account == "" {
		account = strings.TrimSpace(req.Mobile)
	}

	// 步骤 2：基础入参校验。
	if account == "" || req.Password == "" {
		return nil, domain.ErrInvalidCredentials
	}

	// 步骤 3：查询用户并统一“用户不存在/密码错误”为凭证错误，避免用户名探测。
	user, err := s.users.GetByLoginAccount(ctx, account)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	if !verifyPassword(user.Password, req.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	// 步骤 4：生成 access/refresh token。
	accessToken, err := generateToken(32)
	if err != nil {
		return nil, err
	}
	refreshToken, err := generateToken(32)
	if err != nil {
		return nil, err
	}

	// 步骤 5：保存 access token -> userID 映射，用于后续 introspect。
	if err := s.tokens.SetToken(ctx, accessToken, user.ID, AccessTokenTTL); err != nil {
		return nil, err
	}

	// 步骤 6：组装兼容响应（legacy 字段 + camelCase 字段）。
	expiresInSec := int(AccessTokenTTL.Seconds())
	return &vo.LoginResp{
		Token:        accessToken,
		Expire:       expiresInSec,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		Mobile:       user.Mobile,
		Nickname:     user.Nickname,
		ParentID:     user.ParentID,
		Level:        user.Level,
		InviteCode:   user.InviteCode,
		Status:       user.Status,
		Verified:     user.Verified,
		Avatar:       user.Avatar,
		Remark:       user.Remark,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}, nil
}

// Register 创建 users 表用户记录。
func (s *Service) Register(ctx context.Context, req domain.RegisterReq) (*vo.UserResp, error) {
	// 步骤 1：统一入参清洗，避免首尾空格导致脏数据写库。
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	req.Email = strings.TrimSpace(req.Email)
	req.Mobile = strings.TrimSpace(req.Mobile)
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.ParentID = strings.TrimSpace(req.ParentID)
	req.InviteCode = strings.TrimSpace(req.InviteCode)
	req.Avatar = strings.TrimSpace(req.Avatar)
	req.Remark = strings.TrimSpace(req.Remark)

	// 步骤 2：基础必填校验，对齐 users.username/password/mobile 的 NOT NULL 约束。
	if req.Username == "" || req.Password == "" || req.Mobile == "" {
		return nil, domain.ErrInvalidCredentials
	}

	// 步骤 3：用户名唯一性校验。
	_, err := s.users.GetByUsername(ctx, req.Username)
	if err == nil {
		return nil, domain.ErrUsernameExists
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	// 步骤 4：手机号唯一性校验。
	_, err = s.users.GetByMobile(ctx, req.Mobile)
	if err == nil {
		return nil, domain.ErrMobileExists
	}
	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	// 步骤 5：创建用户主记录（密码按历史兼容策略做 MD5 哈希）。
	userID, err := s.users.Create(ctx, req, hashPassword(req.Password))
	if err != nil {
		return nil, err
	}
	return s.GetUserProfile(ctx, userID)
}

// Logout 删除 access token 绑定关系，使当前登录态失效。
func (s *Service) Logout(ctx context.Context, token string) error {
	// 步骤 1：校验 token 非空。
	if strings.TrimSpace(token) == "" {
		return domain.ErrTokenRequired
	}
	// 步骤 2：删除 token 映射。
	return s.tokens.DeleteToken(ctx, token)
}

// IntrospectToken 根据 token 反查 userID，供网关和服务内鉴权复用。
func (s *Service) IntrospectToken(ctx context.Context, token string) (string, error) {
	// 步骤 1：通过 token 仓储反查 userID。
	return s.tokens.GetUserIDByToken(ctx, token)
}

// ResolveUserID 优先信任上游注入的 headerUserID，否则退化为 token 反查。
func (s *Service) ResolveUserID(ctx context.Context, headerUserID, token string) (string, error) {
	// 步骤 1：优先使用上游注入的 header 用户标识。
	if strings.TrimSpace(headerUserID) != "" {
		return strings.TrimSpace(headerUserID), nil
	}

	// 步骤 2：回退为 token 反查。
	return s.IntrospectToken(ctx, token)
}

// GetUserProfile 查询用户资料并映射为对外返回 VO。
func (s *Service) GetUserProfile(ctx context.Context, userID string) (*vo.UserResp, error) {
	// 步骤 1：查询用户实体。
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// 步骤 2：映射为对外 VO。
	return &vo.UserResp{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Mobile:     user.Mobile,
		Nickname:   user.Nickname,
		ParentID:   user.ParentID,
		Level:      user.Level,
		InviteCode: user.InviteCode,
		Status:     user.Status,
		Verified:   user.Verified,
		Avatar:     user.Avatar,
		Remark:     user.Remark,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}, nil
}

// UpdateUserProfile 更新资料后返回最新资料，避免前端二次查询。
func (s *Service) UpdateUserProfile(ctx context.Context, userID string, req domain.UpdateMeReq) (*vo.UserResp, error) {
	// 步骤 1：校验至少有一个更新字段。
	if strings.TrimSpace(req.Email) == "" && strings.TrimSpace(req.Mobile) == "" && strings.TrimSpace(req.Nickname) == "" && strings.TrimSpace(req.Avatar) == "" && strings.TrimSpace(req.Remark) == "" {
		return nil, errors.New("at least one field is required")
	}

	// 步骤 2：执行资料更新。
	if err := s.users.UpdateProfile(ctx, userID, req); err != nil {
		return nil, err
	}

	// 步骤 3：返回更新后最新资料。
	return s.GetUserProfile(ctx, userID)
}

// hashPassword 与 Java 旧系统保持一致，使用 MD5 小写十六进制。
// 注意：这里只为兼容历史系统；新系统建议采用更安全的密码哈希方案。
func hashPassword(raw string) string {
	// Legacy (do not delete): SHA-256 hashing
	// sum := sha256.Sum256([]byte(raw))
	// return hex.EncodeToString(sum[:])

	// 步骤 1：对原始密码做 MD5 并转小写十六进制字符串。
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// verifyPassword 兼容明文遗留数据与 MD5 哈希数据。
func verifyPassword(stored, provided string) bool {
	// 步骤 1：兼容“明文等于”与“MD5 后等于”两种历史数据形态。
	return stored == provided || stored == hashPassword(provided)
}

// generateToken 生成指定字节长度的随机 token（hex 编码后长度翻倍）。
func generateToken(byteLen int) (string, error) {
	// 步骤 1：生成随机字节序列。
	raw := make([]byte, byteLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	// 步骤 2：编码为 hex 字符串返回。
	return hex.EncodeToString(raw), nil
}
