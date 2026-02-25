package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// SemesterRepository 学期数据访问接口
type SemesterRepository interface {
	Create(ctx context.Context, semester *model.Semester) error
	GetByID(ctx context.Context, id string) (*model.Semester, error)
	GetCurrent(ctx context.Context) (*model.Semester, error)
	List(ctx context.Context) ([]model.Semester, error)
	Update(ctx context.Context, semester *model.Semester) error
	Delete(ctx context.Context, id string, deletedBy string) error
	ClearActive(ctx context.Context) error
}

type semesterRepo struct {
	db *gorm.DB
}

// NewSemesterRepo 创建 SemesterRepository 实例
func NewSemesterRepo(db *gorm.DB) SemesterRepository {
	return &semesterRepo{db: db}
}

func (r *semesterRepo) Create(ctx context.Context, semester *model.Semester) error {
	return r.db.WithContext(ctx).Create(semester).Error
}

func (r *semesterRepo) GetByID(ctx context.Context, id string) (*model.Semester, error) {
	var semester model.Semester
	err := r.db.WithContext(ctx).
		Where("semester_id = ?", id).
		First(&semester).Error
	if err != nil {
		return nil, err
	}
	return &semester, nil
}

func (r *semesterRepo) GetCurrent(ctx context.Context) (*model.Semester, error) {
	var semester model.Semester
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		First(&semester).Error
	if err != nil {
		return nil, err
	}
	return &semester, nil
}

func (r *semesterRepo) List(ctx context.Context) ([]model.Semester, error) {
	var semesters []model.Semester
	err := r.db.WithContext(ctx).
		Order("start_date DESC").
		Find(&semesters).Error
	return semesters, err
}

func (r *semesterRepo) Update(ctx context.Context, semester *model.Semester) error {
	return r.db.WithContext(ctx).Save(semester).Error
}

func (r *semesterRepo) Delete(ctx context.Context, id string, deletedBy string) error {
	return r.db.WithContext(ctx).
		Model(&model.Semester{}).
		Where("semester_id = ?", id).
		Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": gorm.Expr("NOW()"),
		}).Error
}

// ClearActive 将所有学期的 is_active 设为 false
func (r *semesterRepo) ClearActive(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Model(&model.Semester{}).
		Where("is_active = ?", true).
		Update("is_active", false).Error
}
