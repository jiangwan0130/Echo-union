package repository

import "gorm.io/gorm"

// Repository 所有 Repository 的聚合入口
type Repository struct {
	User                   UserRepository
	Department             DepartmentRepository
	InviteCode             InviteCodeRepository
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
		User:                   NewUserRepo(db),
		Department:             NewDepartmentRepo(db),
		InviteCode:             NewInviteCodeRepo(db),
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
