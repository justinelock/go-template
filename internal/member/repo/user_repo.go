package repo

import (
	"context"

	"go-template/internal/member/domain"
)

// UserRepository 定义 member 域用户与签到数据访问能力：
// - app 层仅依赖该接口，不绑定具体存储实现；
// - 支持用户资料、默认账户、签到状态/记录等核心读写能力。
type UserRepository interface {
	// GetByUsername 按用户名查询用户记录（登录场景）。
	GetByUsername(ctx context.Context, username string) (*domain.UserRecord, error)
	// GetByMobile 按手机号查询用户记录（登录/注册唯一性校验场景）。
	GetByMobile(ctx context.Context, mobile string) (*domain.UserRecord, error)
	// GetByLoginAccount 按用户名或手机号查询用户记录（登录场景）。
	GetByLoginAccount(ctx context.Context, account string) (*domain.UserRecord, error)
	// GetByID 按用户 ID 查询用户记录（资料场景）。
	GetByID(ctx context.Context, userID string) (*domain.UserRecord, error)
	// Create 创建用户主记录并返回新用户 ID。
	Create(ctx context.Context, req domain.RegisterReq, hashedPassword string) (string, error)
	// UpdateProfile 更新用户资料字段（部分更新）。
	UpdateProfile(ctx context.Context, userID string, req domain.UpdateMeReq) error
}
