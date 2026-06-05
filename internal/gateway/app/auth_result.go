package app

// AuthResult 网关鉴权 introspect 的统一返回结构。
type AuthResult struct {
	// 步骤 1：当前登录用户 ID。
	UserID string
	// 步骤 2：用户角色（user/admin），供 RBAC 与 X-User-Role 注入。
	Role string
}
