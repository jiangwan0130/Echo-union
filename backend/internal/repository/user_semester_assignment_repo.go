package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// UserSemesterAssignmentRepository 用户学期分配数据访问接口
type UserSemesterAssignmentRepository interface {
	Create(ctx context.Context, assignment *model.UserSemesterAssignment) error
	BatchUpsert(ctx context.Context, semesterID string, userIDs []string, dutyRequired bool, callerID string) error
	ListBySemester(ctx context.Context, semesterID string) ([]model.UserSemesterAssignment, error)
	ListByDepartmentAndSemester(ctx context.Context, departmentID, semesterID string) ([]model.UserSemesterAssignment, error)
	ListDutyRequiredSubmitted(ctx context.Context, semesterID string) ([]model.UserSemesterAssignment, error)
	CountDutyRequired(ctx context.Context, semesterID string) (int64, error)
	CountDutyRequiredSubmitted(ctx context.Context, semesterID string) (int64, error)
	GetByUserAndSemester(ctx context.Context, userID, semesterID string) (*model.UserSemesterAssignment, error)
	UpdateTimetableStatus(ctx context.Context, assignmentID string, status string, submittedAt *time.Time, updatedBy string) error
	UpdateDutyRequired(ctx context.Context, assignmentID string, dutyRequired bool, updatedBy string) error
	// ListDutyRequiredBySemester 列出指定学期需要值班的分配记录（含 User + Department）
	ListDutyRequiredBySemester(ctx context.Context, semesterID string) ([]model.UserSemesterAssignment, error)
}

type userSemesterAssignmentRepo struct {
	db *gorm.DB
}

// NewUserSemesterAssignmentRepo 创建 UserSemesterAssignmentRepository 实例
func NewUserSemesterAssignmentRepo(db *gorm.DB) UserSemesterAssignmentRepository {
	return &userSemesterAssignmentRepo{db: db}
}

func (r *userSemesterAssignmentRepo) ListBySemester(ctx context.Context, semesterID string) ([]model.UserSemesterAssignment, error) {
	var assignments []model.UserSemesterAssignment
	err := r.db.WithContext(ctx).
		Preload("User").Preload("User.Department").
		Where("semester_id = ?", semesterID).
		Find(&assignments).Error
	return assignments, err
}

func (r *userSemesterAssignmentRepo) ListDutyRequiredSubmitted(ctx context.Context, semesterID string) ([]model.UserSemesterAssignment, error) {
	var assignments []model.UserSemesterAssignment
	err := r.db.WithContext(ctx).
		Preload("User").Preload("User.Department").
		Where("semester_id = ? AND duty_required = ? AND timetable_status = ?", semesterID, true, "submitted").
		Find(&assignments).Error
	return assignments, err
}

func (r *userSemesterAssignmentRepo) CountDutyRequired(ctx context.Context, semesterID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserSemesterAssignment{}).
		Where("semester_id = ? AND duty_required = ?", semesterID, true).
		Count(&count).Error
	return count, err
}

func (r *userSemesterAssignmentRepo) CountDutyRequiredSubmitted(ctx context.Context, semesterID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserSemesterAssignment{}).
		Where("semester_id = ? AND duty_required = ? AND timetable_status = ?", semesterID, true, "submitted").
		Count(&count).Error
	return count, err
}

func (r *userSemesterAssignmentRepo) GetByUserAndSemester(ctx context.Context, userID, semesterID string) (*model.UserSemesterAssignment, error) {
	var a model.UserSemesterAssignment
	err := r.db.WithContext(ctx).
		Preload("User").Preload("User.Department").
		Where("user_id = ? AND semester_id = ?", userID, semesterID).
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *userSemesterAssignmentRepo) UpdateTimetableStatus(ctx context.Context, assignmentID string, status string, submittedAt *time.Time, updatedBy string) error {
	updates := map[string]interface{}{
		"timetable_status":       status,
		"timetable_submitted_at": submittedAt,
		"updated_by":             updatedBy,
	}
	return r.db.WithContext(ctx).
		Model(&model.UserSemesterAssignment{}).
		Where("assignment_id = ?", assignmentID).
		Updates(updates).Error
}

func (r *userSemesterAssignmentRepo) UpdateDutyRequired(ctx context.Context, assignmentID string, dutyRequired bool, updatedBy string) error {
	updates := map[string]interface{}{
		"duty_required": dutyRequired,
		"updated_by":    updatedBy,
	}
	return r.db.WithContext(ctx).
		Model(&model.UserSemesterAssignment{}).
		Where("assignment_id = ?", assignmentID).
		Updates(updates).Error
}

func (r *userSemesterAssignmentRepo) Create(ctx context.Context, assignment *model.UserSemesterAssignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *userSemesterAssignmentRepo) BatchUpsert(ctx context.Context, semesterID string, userIDs []string, dutyRequired bool, callerID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, uid := range userIDs {
			var existing model.UserSemesterAssignment
			err := tx.Where("user_id = ? AND semester_id = ? AND deleted_at IS NULL", uid, semesterID).First(&existing).Error
			if err == nil {
				// 已存在 → 更新 duty_required
				tx.Model(&existing).Updates(map[string]interface{}{
					"duty_required": dutyRequired,
					"updated_by":    callerID,
				})
			} else {
				// 不存在 → 创建
				a := model.UserSemesterAssignment{
					UserID:       uid,
					SemesterID:   semesterID,
					DutyRequired: dutyRequired,
				}
				a.CreatedBy = &callerID
				a.UpdatedBy = &callerID
				if err := tx.Create(&a).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *userSemesterAssignmentRepo) ListByDepartmentAndSemester(ctx context.Context, departmentID, semesterID string) ([]model.UserSemesterAssignment, error) {
	var assignments []model.UserSemesterAssignment
	err := r.db.WithContext(ctx).
		Preload("User").Preload("User.Department").
		Joins("JOIN users ON users.user_id = user_semester_assignments.user_id AND users.deleted_at IS NULL").
		Where("user_semester_assignments.semester_id = ? AND users.department_id = ? AND user_semester_assignments.deleted_at IS NULL", semesterID, departmentID).
		Find(&assignments).Error
	return assignments, err
}

func (r *userSemesterAssignmentRepo) ListDutyRequiredBySemester(ctx context.Context, semesterID string) ([]model.UserSemesterAssignment, error) {
	var assignments []model.UserSemesterAssignment
	err := r.db.WithContext(ctx).
		Preload("User").Preload("User.Department").
		Where("semester_id = ? AND duty_required = ?", semesterID, true).
		Find(&assignments).Error
	return assignments, err
}
