package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// SystemConfigRepository 系统配置数据访问接口
type SystemConfigRepository interface {
	Get(ctx context.Context) (*model.SystemConfig, error)
	Update(ctx context.Context, cfg *model.SystemConfig) error
}

type systemConfigRepo struct {
	db *gorm.DB
}

// NewSystemConfigRepo 创建 SystemConfigRepository 实例
func NewSystemConfigRepo(db *gorm.DB) SystemConfigRepository {
	return &systemConfigRepo{db: db}
}

func (r *systemConfigRepo) Get(ctx context.Context) (*model.SystemConfig, error) {
	var cfg model.SystemConfig
	err := r.db.WithContext(ctx).First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *systemConfigRepo) Update(ctx context.Context, cfg *model.SystemConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}
