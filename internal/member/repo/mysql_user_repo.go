package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-template/internal/member/domain"
)

// MySQLUserRepo 基于 MySQL users 表的用户仓储实现。
type MySQLUserRepo struct {
	// 步骤 1：MySQL 连接池。
	db *sql.DB
}

func NewMySQLUserRepo(db *sql.DB) *MySQLUserRepo {
	// 步骤 1：注入 MySQL 连接并返回仓储实例。
	return &MySQLUserRepo{db: db}
}

// GetByUsername 按用户名查询用户信息。
func (r *MySQLUserRepo) GetByUsername(ctx context.Context, username string) (*domain.UserRecord, error) {
	// 步骤 1：构造单行查询并映射为领域用户对象。
	row := r.db.QueryRowContext(ctx, selectUserSQL("username = ?"), strings.TrimSpace(username))
	return scanUser(row)
}

// GetByMobile 按手机号查询用户信息。
func (r *MySQLUserRepo) GetByMobile(ctx context.Context, mobile string) (*domain.UserRecord, error) {
	// 步骤 1：构造单行查询并映射为领域用户对象。
	row := r.db.QueryRowContext(ctx, selectUserSQL("mobile = ?"), strings.TrimSpace(mobile))
	return scanUser(row)
}

// GetByLoginAccount 按用户名或手机号查询用户信息。
func (r *MySQLUserRepo) GetByLoginAccount(ctx context.Context, account string) (*domain.UserRecord, error) {
	// 步骤 1：登录入口兼容 username/mobile，均来自 users 表唯一字段。
	account = strings.TrimSpace(account)
	row := r.db.QueryRowContext(ctx, selectUserSQL("username = ? OR mobile = ?"), account, account)
	return scanUser(row)
}

// GetByID 按用户 ID 查询用户信息。
func (r *MySQLUserRepo) GetByID(ctx context.Context, userID string) (*domain.UserRecord, error) {
	// 步骤 1：使用 CAST 兼容字符串 userID 入参。
	row := r.db.QueryRowContext(ctx, selectUserSQL("id = CAST(? AS UNSIGNED)"), strings.TrimSpace(userID))
	return scanUser(row)
}

// Create 写入用户主记录并返回新用户 ID。
func (r *MySQLUserRepo) Create(ctx context.Context, req domain.RegisterReq, hashedPassword string) (string, error) {
	// 步骤 1：生成统一时间戳并插入用户主表。
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO users(
			username, password, email, mobile, nickname, parent_id, invite_code,
			avatar, remark, role, created_at, updated_at
		)
		VALUES (?, ?, NULLIF(?,''), ?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), 'user', ?, ?)
	`, req.Username, hashedPassword, req.Email, req.Mobile, req.Nickname, req.ParentID, req.InviteCode, req.Avatar, req.Remark, now, now)
	if err != nil {
		return "", err
	}

	// 步骤 2：读取自增主键并转成字符串返回。
	id, err := res.LastInsertId()
	if err != nil {
		return "", fmt.Errorf("get inserted user id failed: %w", err)
	}
	return strconv.FormatInt(id, 10), nil
}

// UpdateProfile 更新用户资料字段（部分更新）。
func (r *MySQLUserRepo) UpdateProfile(ctx context.Context, userID string, req domain.UpdateMeReq) error {
	// 步骤 1：对入参做 trim 后执行条件更新。
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET email = COALESCE(NULLIF(?, ''), email),
		    mobile = COALESCE(NULLIF(?, ''), mobile),
		    nickname = COALESCE(NULLIF(?, ''), nickname),
		    avatar = COALESCE(NULLIF(?, ''), avatar),
		    remark = COALESCE(NULLIF(?, ''), remark),
		    updated_at = ?
		WHERE id = CAST(? AS UNSIGNED)
	`, strings.TrimSpace(req.Email), strings.TrimSpace(req.Mobile), strings.TrimSpace(req.Nickname), strings.TrimSpace(req.Avatar), strings.TrimSpace(req.Remark), time.Now().UTC(), userID)
	return err
}

func (r *MySQLUserRepo) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET password = ?, updated_at = ? WHERE id = CAST(? AS UNSIGNED)
	`, hashedPassword, time.Now().UTC(), userID)
	return err
}

func selectUserSQL(where string) string {
	return fmt.Sprintf(`
		SELECT CAST(id AS CHAR), username, password,
		       COALESCE(email,''), mobile, COALESCE(nickname,''),
		       COALESCE(CAST(parent_id AS CHAR), ''), level, COALESCE(invite_code,''),
		       COALESCE(status, 0), verified, COALESCE(avatar,''), COALESCE(remark,''), COALESCE(role,'user'),
		       DATE_FORMAT(created_at, '%%Y-%%m-%%d %%H:%%i:%%s'),
		       COALESCE(DATE_FORMAT(updated_at, '%%Y-%%m-%%d %%H:%%i:%%s'), '')
		FROM users
		WHERE %s
		LIMIT 1
	`, where)
}

// scanUser 将 SQL 单行结果映射为领域用户对象。
func scanUser(row *sql.Row) (*domain.UserRecord, error) {
	// 步骤 1：扫描用户字段。
	var user domain.UserRecord
	var verified int
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Email,
		&user.Mobile,
		&user.Nickname,
		&user.ParentID,
		&user.Level,
		&user.InviteCode,
		&user.Status,
		&verified,
		&user.Avatar,
		&user.Remark,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		// 步骤 2：无记录统一映射为领域错误。
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		// 步骤 3：数据库扫描异常原样返回。
		return nil, err
	}

	// 步骤 4：转换 tinyint 认证状态并返回完整用户对象。
	user.Verified = verified != 0
	return &user, nil
}
