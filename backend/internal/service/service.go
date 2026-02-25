package service

import (
	"go.uber.org/zap"

	"echo-union/backend/config"
	"echo-union/backend/internal/repository"
	"echo-union/backend/pkg/jwt"
	"echo-union/backend/pkg/redis"
)

// Service 所有 Service 的聚合入口
type Service struct {
	Auth         AuthService
	User         UserService
	Department   DepartmentService
	Semester     SemesterService
	TimeSlot     TimeSlotService
	Location     LocationService
	SystemConfig SystemConfigService
	ScheduleRule ScheduleRuleService
	Schedule     ScheduleService
	Timetable    TimetableService
	Export       ExportService
}

// NewService 创建 Service 聚合
func NewService(
	cfg *config.Config,
	repo *repository.Repository,
	jwtMgr *jwt.Manager,
	rdb *redis.Client,
	logger *zap.Logger,
) *Service {
	return &Service{
		Auth:         NewAuthService(cfg, repo, jwtMgr, rdb, logger),
		User:         NewUserService(repo, logger),
		Department:   NewDepartmentService(repo, logger),
		Semester:     NewSemesterService(repo, logger),
		TimeSlot:     NewTimeSlotService(repo, logger),
		Location:     NewLocationService(repo, logger),
		SystemConfig: NewSystemConfigService(repo, logger),
		ScheduleRule: NewScheduleRuleService(repo, logger),
		Schedule:     NewScheduleService(repo, logger),
		Timetable:    NewTimetableService(repo, logger),
		Export:       NewExportService(repo, logger),
	}
}
