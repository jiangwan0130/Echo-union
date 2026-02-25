package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// TimeSlotRepository 时间段数据访问接口
type TimeSlotRepository interface {
	Create(ctx context.Context, slot *model.TimeSlot) error
	GetByID(ctx context.Context, id string) (*model.TimeSlot, error)
	List(ctx context.Context, semesterID string, dayOfWeek *int) ([]model.TimeSlot, error)
	Update(ctx context.Context, slot *model.TimeSlot) error
	Delete(ctx context.Context, id string, deletedBy string) error
}

type timeSlotRepo struct {
	db *gorm.DB
}

// NewTimeSlotRepo 创建 TimeSlotRepository 实例
func NewTimeSlotRepo(db *gorm.DB) TimeSlotRepository {
	return &timeSlotRepo{db: db}
}

func (r *timeSlotRepo) Create(ctx context.Context, slot *model.TimeSlot) error {
	return r.db.WithContext(ctx).Create(slot).Error
}

func (r *timeSlotRepo) GetByID(ctx context.Context, id string) (*model.TimeSlot, error) {
	var slot model.TimeSlot
	err := r.db.WithContext(ctx).
		Preload("Semester").
		Where("time_slot_id = ?", id).
		First(&slot).Error
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

func (r *timeSlotRepo) List(ctx context.Context, semesterID string, dayOfWeek *int) ([]model.TimeSlot, error) {
	var slots []model.TimeSlot
	db := r.db.WithContext(ctx).Where("is_active = ?", true)

	if semesterID != "" {
		db = db.Where("(semester_id = ? OR semester_id IS NULL)", semesterID)
	}
	if dayOfWeek != nil {
		db = db.Where("day_of_week = ?", *dayOfWeek)
	}

	err := db.Preload("Semester").
		Order("day_of_week ASC, start_time ASC").
		Find(&slots).Error
	return slots, err
}

func (r *timeSlotRepo) Update(ctx context.Context, slot *model.TimeSlot) error {
	return r.db.WithContext(ctx).Save(slot).Error
}

func (r *timeSlotRepo) Delete(ctx context.Context, id string, deletedBy string) error {
	return r.db.WithContext(ctx).
		Model(&model.TimeSlot{}).
		Where("time_slot_id = ?", id).
		Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": gorm.Expr("NOW()"),
		}).Error
}
