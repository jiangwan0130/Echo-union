package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// InviteCodeRepository 邀请码数据访问接口
type InviteCodeRepository interface {
	Create(ctx context.Context, code *model.InviteCode) error
	GetByCode(ctx context.Context, code string) (*model.InviteCode, error)
	// GetByCodeForUpdate 使用 SELECT ... FOR UPDATE 行级锁查询邀请码，防止并发使用
	GetByCodeForUpdate(ctx context.Context, code string) (*model.InviteCode, error)
	MarkUsed(ctx context.Context, inviteCodeID, userID string) error
}

type inviteCodeRepo struct {
	db *gorm.DB
}

// NewInviteCodeRepo 创建 InviteCodeRepository 实例
func NewInviteCodeRepo(db *gorm.DB) InviteCodeRepository {
	return &inviteCodeRepo{db: db}
}

func (r *inviteCodeRepo) Create(ctx context.Context, code *model.InviteCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

// GetByCode 根据邀请码字符串查询（仅返回未软删除的记录）
func (r *inviteCodeRepo) GetByCode(ctx context.Context, code string) (*model.InviteCode, error) {
	var invite model.InviteCode
	err := r.db.WithContext(ctx).
		Where("code = ?", code).
		First(&invite).Error
	if err != nil {
		return nil, err
	}
	return &invite, nil
}

// GetByCodeForUpdate 使用 SELECT ... FOR UPDATE 行级锁查询邀请码
// 必须在已有事务的 *gorm.DB 上调用（通过 Repository.WithTx 注入事务连接）
func (r *inviteCodeRepo) GetByCodeForUpdate(ctx context.Context, code string) (*model.InviteCode, error) {
	var invite model.InviteCode
	err := r.db.WithContext(ctx).
		Set("gorm:query_option", "FOR UPDATE").
		Where("code = ?", code).
		First(&invite).Error
	if err != nil {
		return nil, err
	}
	return &invite, nil
}

// MarkUsed 标记邀请码为已使用
func (r *inviteCodeRepo) MarkUsed(ctx context.Context, inviteCodeID, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.InviteCode{}).
		Where("invite_code_id = ?", inviteCodeID).
		Updates(map[string]interface{}{
			"used_at":    now,
			"used_by":    userID,
			"updated_at": now,
			"updated_by": userID,
		}).Error
}
