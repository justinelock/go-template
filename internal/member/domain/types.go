package domain

import (
	"errors"
)

// LoginReq 登录请求（兼容 username/mobile 双入口）。
type LoginReq struct {
	// Username 为用户名登录入口。
	Username string
	// Mobile 为手机号登录入口（username 为空时使用）。
	Mobile string
	// Password 为登录密码明文（由 app 层进行校验）。
	Password string
}

// RegisterReq 注册请求。
type RegisterReq struct {
	// Username 为注册用户名。
	Username string
	// Password 为注册密码明文。
	Password string
	// Email 为邮箱（可选）。
	Email string
	// Mobile 为手机号。
	Mobile string
	// Nickname 为昵称（可选）。
	Nickname string
	// ParentID 为上级代理 ID（可选）。
	ParentID string
	// InviteCode 为邀请码（可选）。
	InviteCode string
	// Avatar 为用户头像（可选）。
	Avatar string
	// Remark 为说明（可选）。
	Remark string
}

// UpdateMeReq 更新用户资料请求。
type UpdateMeReq struct {
	// Email 为用户邮箱。
	Email string
	// Mobile 为手机号。
	Mobile string
	// Nickname 为昵称。
	Nickname string
	// Avatar 为头像地址或标识。
	Avatar string
	// Remark 为说明。
	Remark string
}

// UserRecord 用户实体快照（repo 层读写模型）。
type UserRecord struct {
	// ID 为用户主键 ID。
	ID string
	// Username 为用户名。
	Username string
	// Password 为持久化密码（可能是哈希值）。
	Password string
	// Email 为邮箱。
	Email string
	// Mobile 为手机号。
	Mobile string
	// Nickname 为昵称。
	Nickname string
	// ParentID 为上级代理 ID。
	ParentID string
	// Level 为代理等级。
	Level int
	// InviteCode 为邀请码。
	InviteCode string
	// Status 为用户状态。
	Status int
	// Verified 为是否已实名认证。
	Verified bool
	// Avatar 为头像。
	Avatar string
	// Remark 为说明。
	Remark string
	// CreatedAt 为创建时间。
	CreatedAt string
	// UpdatedAt 为更新时间。
	UpdatedAt string
}

var (
	// ErrUsernameExists 表示用户名已存在。
	ErrUsernameExists = errors.New("username already exists")
	// ErrMobileExists 表示手机号已存在。
	ErrMobileExists = errors.New("mobile already exists")
	// ErrInvalidCredentials 表示账号或密码错误。
	ErrInvalidCredentials = errors.New("username or password is invalid")
	// ErrTokenRequired 表示缺少 token。
	ErrTokenRequired = errors.New("token is required")
	// ErrTokenInvalid 表示 token 无效或已过期。
	ErrTokenInvalid = errors.New("token is invalid or expired")
	// ErrUserNotFound 表示用户不存在。
	ErrUserNotFound = errors.New("user not found")
)
