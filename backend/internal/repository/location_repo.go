package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// LocationRepository 地点数据访问接口
type LocationRepository interface {
	Create(ctx context.Context, loc *model.Location) error
	GetByID(ctx context.Context, id string) (*model.Location, error)
	List(ctx context.Context, includeInactive bool) ([]model.Location, error)
	Update(ctx context.Context, loc *model.Location) error
	Delete(ctx context.Context, id string, deletedBy string) error
}

type locationRepo struct {
	db *gorm.DB
}

// NewLocationRepo 创建 LocationRepository 实例
func NewLocationRepo(db *gorm.DB) LocationRepository {
	return &locationRepo{db: db}
}

func (r *locationRepo) Create(ctx context.Context, loc *model.Location) error {
	return r.db.WithContext(ctx).Create(loc).Error
}

func (r *locationRepo) GetByID(ctx context.Context, id string) (*model.Location, error) {
	var loc model.Location
	err := r.db.WithContext(ctx).
		Where("location_id = ?", id).
		First(&loc).Error
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *locationRepo) List(ctx context.Context, includeInactive bool) ([]model.Location, error) {
	var locations []model.Location
	db := r.db.WithContext(ctx)

	if !includeInactive {
		db = db.Where("is_active = ?", true)
	}

	err := db.Order("is_default DESC, name ASC").Find(&locations).Error
	return locations, err
}

func (r *locationRepo) Update(ctx context.Context, loc *model.Location) error {
	return r.db.WithContext(ctx).Save(loc).Error
}

func (r *locationRepo) Delete(ctx context.Context, id string, deletedBy string) error {
	return r.db.WithContext(ctx).
		Model(&model.Location{}).
		Where("location_id = ?", id).
		Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": gorm.Expr("NOW()"),
		}).Error
}
