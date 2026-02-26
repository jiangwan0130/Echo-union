package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
	pkgerrors "echo-union/backend/pkg/errors"
)

// ScheduleRepository 排班表数据访问接口
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *model.Schedule) error
	GetByID(ctx context.Context, id string) (*model.Schedule, error)
	GetBySemester(ctx context.Context, semesterID string) (*model.Schedule, error)
	GetLatestBySemester(ctx context.Context, semesterID string) (*model.Schedule, error)
	Update(ctx context.Context, schedule *model.Schedule) error
	Delete(ctx context.Context, id string) error
}

// ScheduleItemRepository 排班明细数据访问接口
type ScheduleItemRepository interface {
	BatchCreate(ctx context.Context, items []model.ScheduleItem) error
	GetByID(ctx context.Context, id string) (*model.ScheduleItem, error)
	ListBySchedule(ctx context.Context, scheduleID string) ([]model.ScheduleItem, error)
	ListByScheduleAndMember(ctx context.Context, scheduleID, memberID string) ([]model.ScheduleItem, error)
	Update(ctx context.Context, item *model.ScheduleItem) error
	DeleteBySchedule(ctx context.Context, scheduleID string) error
}

// ScheduleMemberSnapshotRepository 排班成员快照数据访问接口
type ScheduleMemberSnapshotRepository interface {
	BatchCreate(ctx context.Context, snapshots []model.ScheduleMemberSnapshot) error
	ListBySchedule(ctx context.Context, scheduleID string) ([]model.ScheduleMemberSnapshot, error)
	DeleteBySchedule(ctx context.Context, scheduleID string) error
}

// ScheduleChangeLogRepository 排班变更日志数据访问接口
type ScheduleChangeLogRepository interface {
	Create(ctx context.Context, log *model.ScheduleChangeLog) error
	ListBySchedule(ctx context.Context, scheduleID string, offset, limit int) ([]model.ScheduleChangeLog, int64, error)
}

// ── Schedule Repository 实现 ──

type scheduleRepo struct {
	db *gorm.DB
}

func NewScheduleRepo(db *gorm.DB) ScheduleRepository {
	return &scheduleRepo{db: db}
}

func (r *scheduleRepo) Create(ctx context.Context, schedule *model.Schedule) error {
	return r.db.WithContext(ctx).Create(schedule).Error
}

func (r *scheduleRepo) GetByID(ctx context.Context, id string) (*model.Schedule, error) {
	var schedule model.Schedule
	err := r.db.WithContext(ctx).
		Preload("Semester").
		Where("schedule_id = ?", id).
		First(&schedule).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *scheduleRepo) GetBySemester(ctx context.Context, semesterID string) (*model.Schedule, error) {
	var schedule model.Schedule
	err := r.db.WithContext(ctx).
		Preload("Semester").
		Where("semester_id = ? AND status != ?", semesterID, "archived").
		Order("created_at DESC").
		First(&schedule).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *scheduleRepo) GetLatestBySemester(ctx context.Context, semesterID string) (*model.Schedule, error) {
	var schedule model.Schedule
	err := r.db.WithContext(ctx).
		Preload("Semester").
		Where("semester_id = ?", semesterID).
		Order("created_at DESC").
		First(&schedule).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *scheduleRepo) Update(ctx context.Context, schedule *model.Schedule) error {
	oldVersion := schedule.Version
	result := r.db.WithContext(ctx).
		Model(schedule).
		Where("schedule_id = ? AND version = ?", schedule.ScheduleID, oldVersion).
		Updates(map[string]interface{}{
			"semester_id":  schedule.SemesterID,
			"status":       schedule.Status,
			"published_at": schedule.PublishedAt,
			"updated_by":   schedule.UpdatedBy,
			"version":      oldVersion + 1,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return pkgerrors.ErrOptimisticLock
	}
	schedule.Version = oldVersion + 1
	return nil
}

func (r *scheduleRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("schedule_id = ?", id).
		Delete(&model.Schedule{}).Error
}

// ── ScheduleItem Repository 实现 ──

type scheduleItemRepo struct {
	db *gorm.DB
}

func NewScheduleItemRepo(db *gorm.DB) ScheduleItemRepository {
	return &scheduleItemRepo{db: db}
}

func (r *scheduleItemRepo) BatchCreate(ctx context.Context, items []model.ScheduleItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

func (r *scheduleItemRepo) GetByID(ctx context.Context, id string) (*model.ScheduleItem, error) {
	var item model.ScheduleItem
	err := r.db.WithContext(ctx).
		Preload("TimeSlot").
		Preload("Member").Preload("Member.Department").
		Preload("Location").
		Where("schedule_item_id = ?", id).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *scheduleItemRepo) ListBySchedule(ctx context.Context, scheduleID string) ([]model.ScheduleItem, error) {
	var items []model.ScheduleItem
	err := r.db.WithContext(ctx).
		Preload("TimeSlot").
		Preload("Member").Preload("Member.Department").
		Preload("Location").
		Where("schedule_id = ?", scheduleID).
		Order("week_number ASC, time_slot_id ASC").
		Find(&items).Error
	return items, err
}

func (r *scheduleItemRepo) ListByScheduleAndMember(ctx context.Context, scheduleID, memberID string) ([]model.ScheduleItem, error) {
	var items []model.ScheduleItem
	err := r.db.WithContext(ctx).
		Preload("TimeSlot").
		Preload("Member").Preload("Member.Department").
		Preload("Location").
		Where("schedule_id = ? AND member_id = ?", scheduleID, memberID).
		Order("week_number ASC, time_slot_id ASC").
		Find(&items).Error
	return items, err
}

func (r *scheduleItemRepo) Update(ctx context.Context, item *model.ScheduleItem) error {
	oldVersion := item.Version
	result := r.db.WithContext(ctx).
		Model(item).
		Where("schedule_item_id = ? AND version = ?", item.ScheduleItemID, oldVersion).
		Updates(map[string]interface{}{
			"week_number":  item.WeekNumber,
			"time_slot_id": item.TimeSlotID,
			"member_id":    item.MemberID,
			"location_id":  item.LocationID,
			"updated_by":   item.UpdatedBy,
			"version":      oldVersion + 1,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return pkgerrors.ErrOptimisticLock
	}
	item.Version = oldVersion + 1
	return nil
}

func (r *scheduleItemRepo) DeleteBySchedule(ctx context.Context, scheduleID string) error {
	return r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Delete(&model.ScheduleItem{}).Error
}

// ── ScheduleMemberSnapshot Repository 实现 ──

type scheduleMemberSnapshotRepo struct {
	db *gorm.DB
}

func NewScheduleMemberSnapshotRepo(db *gorm.DB) ScheduleMemberSnapshotRepository {
	return &scheduleMemberSnapshotRepo{db: db}
}

func (r *scheduleMemberSnapshotRepo) BatchCreate(ctx context.Context, snapshots []model.ScheduleMemberSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&snapshots).Error
}

func (r *scheduleMemberSnapshotRepo) ListBySchedule(ctx context.Context, scheduleID string) ([]model.ScheduleMemberSnapshot, error) {
	var snapshots []model.ScheduleMemberSnapshot
	err := r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Find(&snapshots).Error
	return snapshots, err
}

func (r *scheduleMemberSnapshotRepo) DeleteBySchedule(ctx context.Context, scheduleID string) error {
	return r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Delete(&model.ScheduleMemberSnapshot{}).Error
}

// ── ScheduleChangeLog Repository 实现 ──

type scheduleChangeLogRepo struct {
	db *gorm.DB
}

func NewScheduleChangeLogRepo(db *gorm.DB) ScheduleChangeLogRepository {
	return &scheduleChangeLogRepo{db: db}
}

func (r *scheduleChangeLogRepo) Create(ctx context.Context, log *model.ScheduleChangeLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *scheduleChangeLogRepo) ListBySchedule(ctx context.Context, scheduleID string, offset, limit int) ([]model.ScheduleChangeLog, int64, error) {
	var logs []model.ScheduleChangeLog
	var total int64

	db := r.db.WithContext(ctx).Model(&model.ScheduleChangeLog{}).
		Where("schedule_id = ?", scheduleID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, total, err
}
