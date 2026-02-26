package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// UnavailableTimeRepository 不可用时间数据访问接口
type UnavailableTimeRepository interface {
	ListByUserAndSemester(ctx context.Context, userID, semesterID string) ([]model.UnavailableTime, error)
	ListBySemester(ctx context.Context, semesterID string) ([]model.UnavailableTime, error)
	GetByID(ctx context.Context, id string) (*model.UnavailableTime, error)
	Create(ctx context.Context, ut *model.UnavailableTime) error
	Update(ctx context.Context, ut *model.UnavailableTime) error
	Delete(ctx context.Context, id string, deletedBy string) error
}

type unavailableTimeRepo struct {
	db *gorm.DB
}

// NewUnavailableTimeRepo 创建 UnavailableTimeRepository 实例
func NewUnavailableTimeRepo(db *gorm.DB) UnavailableTimeRepository {
	return &unavailableTimeRepo{db: db}
}

func (r *unavailableTimeRepo) ListByUserAndSemester(ctx context.Context, userID, semesterID string) ([]model.UnavailableTime, error) {
	var times []model.UnavailableTime
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND semester_id = ?", userID, semesterID).
		Order("day_of_week ASC, start_time ASC").
		Find(&times).Error
	return times, err
}

func (r *unavailableTimeRepo) ListBySemester(ctx context.Context, semesterID string) ([]model.UnavailableTime, error) {
	var times []model.UnavailableTime
	err := r.db.WithContext(ctx).
		Where("semester_id = ?", semesterID).
		Order("user_id ASC, day_of_week ASC, start_time ASC").
		Find(&times).Error
	return times, err
}

func (r *unavailableTimeRepo) GetByID(ctx context.Context, id string) (*model.UnavailableTime, error) {
	var ut model.UnavailableTime
	err := r.db.WithContext(ctx).Where("unavailable_time_id = ?", id).First(&ut).Error
	if err != nil {
		return nil, err
	}
	return &ut, nil
}

func (r *unavailableTimeRepo) Create(ctx context.Context, ut *model.UnavailableTime) error {
	return r.db.WithContext(ctx).Create(ut).Error
}

func (r *unavailableTimeRepo) Update(ctx context.Context, ut *model.UnavailableTime) error {
	return r.db.WithContext(ctx).Save(ut).Error
}

func (r *unavailableTimeRepo) Delete(ctx context.Context, id string, deletedBy string) error {
	return r.db.WithContext(ctx).
		Model(&model.UnavailableTime{}).
		Where("unavailable_time_id = ?", id).
		Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": gorm.Expr("NOW()"),
		}).Error
}
