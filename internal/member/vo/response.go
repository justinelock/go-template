package vo

// LoginResp 登录响应（兼容旧字段 + 新字段）。
type LoginResp struct {
	// Token 为历史客户端使用的 token 字段。
	Token string `json:"token"`
	// Expire 为历史客户端使用的过期秒数字段。
	Expire int `json:"expire"`

	// AccessToken 为新客户端 access token。
	AccessToken string `json:"accessToken"`
	// RefreshToken 为新客户端 refresh token。
	RefreshToken string `json:"refreshToken"`
	// ID 为登录用户 ID，对应 users.id。
	ID string `json:"id"`
	// Username 为登录用户名。
	Username string `json:"username"`
	// Email 为邮箱。
	Email string `json:"email"`
	// Mobile 为手机号。
	Mobile string `json:"mobile"`
	// Nickname 为昵称。
	Nickname string `json:"nickname"`
	// ParentID 为上级代理 ID。
	ParentID string `json:"parentId"`
	// Level 为代理等级。
	Level int `json:"level"`
	// InviteCode 为邀请码。
	InviteCode string `json:"inviteCode"`
	// Status 为用户状态。
	Status int `json:"status"`
	// Verified 为是否已实名认证。
	Verified bool `json:"verified"`
	// Avatar 为头像。
	Avatar string `json:"avatar"`
	// Remark 为说明。
	Remark string `json:"remark"`
	// CreatedAt 为创建时间。
	CreatedAt string `json:"createdAt"`
	// UpdatedAt 为更新时间。
	UpdatedAt string `json:"updatedAt"`
	// Role 为用户角色。
	Role string `json:"role"`
}

// UserResp 用户资料响应，对应 users 表字段。
type UserResp struct {
	// ID 为用户 ID。
	ID string `json:"id"`
	// Username 为用户名。
	Username string `json:"username"`
	// Email 为邮箱。
	Email string `json:"email"`
	// Mobile 为手机号。
	Mobile string `json:"mobile"`
	// Nickname 为昵称。
	Nickname string `json:"nickname"`
	// ParentID 为上级代理 ID。
	ParentID string `json:"parentId"`
	// Level 为代理等级。
	Level int `json:"level"`
	// InviteCode 为邀请码。
	InviteCode string `json:"inviteCode"`
	// Status 为用户状态。
	Status int `json:"status"`
	// Verified 为是否已实名认证。
	Verified bool `json:"verified"`
	// Avatar 为头像。
	Avatar string `json:"avatar"`
	// Remark 为说明。
	Remark string `json:"remark"`
	// CreatedAt 为创建时间。
	CreatedAt string `json:"createdAt"`
	// UpdatedAt 为更新时间。
	UpdatedAt string `json:"updatedAt"`
	// Role 为用户角色。
	Role string `json:"role"`
}
