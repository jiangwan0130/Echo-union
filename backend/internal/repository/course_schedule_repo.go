package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// CourseScheduleRepository 课表数据访问接口
type CourseScheduleRepository interface {
	ListByUserAndSemester(ctx context.Context, userID, semesterID string) ([]model.CourseSchedule, error)
	ListBySemester(ctx context.Context, semesterID string) ([]model.CourseSchedule, error)
	BatchCreate(ctx context.Context, courses []model.CourseSchedule) error
	DeleteByUserAndSemester(ctx context.Context, userID, semesterID string) error
	// ReplaceByUserAndSemester 在事务中全量替换用户课表：先删除旧数据，再批量插入新数据
	ReplaceByUserAndSemester(ctx context.Context, userID, semesterID string, courses []model.CourseSchedule) error
}

type courseScheduleRepo struct {
	db *gorm.DB
}

// NewCourseScheduleRepo 创建 CourseScheduleRepository 实例
func NewCourseScheduleRepo(db *gorm.DB) CourseScheduleRepository {
	return &courseScheduleRepo{db: db}
}

func (r *courseScheduleRepo) ListByUserAndSemester(ctx context.Context, userID, semesterID string) ([]model.CourseSchedule, error) {
	var courses []model.CourseSchedule
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND semester_id = ?", userID, semesterID).
		Order("day_of_week ASC, start_time ASC").
		Find(&courses).Error
	return courses, err
}

func (r *courseScheduleRepo) ListBySemester(ctx context.Context, semesterID string) ([]model.CourseSchedule, error) {
	var courses []model.CourseSchedule
	err := r.db.WithContext(ctx).
		Where("semester_id = ?", semesterID).
		Order("user_id ASC, day_of_week ASC, start_time ASC").
		Find(&courses).Error
	return courses, err
}

func (r *courseScheduleRepo) BatchCreate(ctx context.Context, courses []model.CourseSchedule) error {
	if len(courses) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&courses).Error
}

func (r *courseScheduleRepo) DeleteByUserAndSemester(ctx context.Context, userID, semesterID string) error {
	// 硬删除：课表替换场景无需保留旧数据
	return r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND semester_id = ?", userID, semesterID).
		Delete(&model.CourseSchedule{}).Error
}

func (r *courseScheduleRepo) ReplaceByUserAndSemester(ctx context.Context, userID, semesterID string, courses []model.CourseSchedule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 硬删除旧课表（替换场景，无需软删除审计）
		if err := tx.Unscoped().Where("user_id = ? AND semester_id = ?", userID, semesterID).
			Delete(&model.CourseSchedule{}).Error; err != nil {
			return err
		}
		if len(courses) > 0 {
			if err := tx.Create(&courses).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
