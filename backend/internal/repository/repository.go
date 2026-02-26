package repository

import (
	"context"

	"gorm.io/gorm"
)

// Repository 所有 Repository 的聚合入口
type Repository struct {
	db                     *gorm.DB
	User                   UserRepository
	Department             DepartmentRepository
	Semester               SemesterRepository
	TimeSlot               TimeSlotRepository
	Location               LocationRepository
	SystemConfig           SystemConfigRepository
	ScheduleRule           ScheduleRuleRepository
	CourseSchedule         CourseScheduleRepository
	UnavailableTime        UnavailableTimeRepository
	UserSemesterAssignment UserSemesterAssignmentRepository
	Schedule               ScheduleRepository
	ScheduleItem           ScheduleItemRepository
	ScheduleMemberSnapshot ScheduleMemberSnapshotRepository
	ScheduleChangeLog      ScheduleChangeLogRepository
}

// NewRepository 创建 Repository 聚合
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db:                     db,
		User:                   NewUserRepo(db),
		Department:             NewDepartmentRepo(db),
		Semester:               NewSemesterRepo(db),
		TimeSlot:               NewTimeSlotRepo(db),
		Location:               NewLocationRepo(db),
		SystemConfig:           NewSystemConfigRepo(db),
		ScheduleRule:           NewScheduleRuleRepo(db),
		CourseSchedule:         NewCourseScheduleRepo(db),
		UnavailableTime:        NewUnavailableTimeRepo(db),
		UserSemesterAssignment: NewUserSemesterAssignmentRepo(db),
		Schedule:               NewScheduleRepo(db),
		ScheduleItem:           NewScheduleItemRepo(db),
		ScheduleMemberSnapshot: NewScheduleMemberSnapshotRepo(db),
		ScheduleChangeLog:      NewScheduleChangeLogRepo(db),
	}
}

// BeginTx 开启数据库事务，返回 *gorm.DB 事务对象。
// 调用方负责在结束时调用 tx.Commit() 或 tx.Rollback()。
// 当 db 为 nil（测试环境）时返回 nil tx，配合 WithTx 使用。
func (r *Repository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	if r.db == nil {
		return nil, nil
	}
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}

// WithTx 使用给定的事务 *gorm.DB 创建一组新的 Repository 实例。
// 事务内的所有操作都将共享同一个数据库连接，保证原子性。
// 当 tx 为 nil（测试环境）时返回自身，mock 不需要真正的事务。
func (r *Repository) WithTx(tx *gorm.DB) *Repository {
	if tx == nil {
		return r
	}
	return &Repository{
		db:                     tx,
		User:                   NewUserRepo(tx),
		Department:             NewDepartmentRepo(tx),
		Semester:               NewSemesterRepo(tx),
		TimeSlot:               NewTimeSlotRepo(tx),
		Location:               NewLocationRepo(tx),
		SystemConfig:           NewSystemConfigRepo(tx),
		ScheduleRule:           NewScheduleRuleRepo(tx),
		CourseSchedule:         NewCourseScheduleRepo(tx),
		UnavailableTime:        NewUnavailableTimeRepo(tx),
		UserSemesterAssignment: NewUserSemesterAssignmentRepo(tx),
		Schedule:               NewScheduleRepo(tx),
		ScheduleItem:           NewScheduleItemRepo(tx),
		ScheduleMemberSnapshot: NewScheduleMemberSnapshotRepo(tx),
		ScheduleChangeLog:      NewScheduleChangeLogRepo(tx),
	}
}
