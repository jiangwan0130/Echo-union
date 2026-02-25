package handler

import (
	"echo-union/backend/config"
	"echo-union/backend/internal/service"
)

// Handler 所有 Handler 的聚合入口
type Handler struct {
	Auth         *AuthHandler
	User         *UserHandler
	Department   *DepartmentHandler
	Semester     *SemesterHandler
	TimeSlot     *TimeSlotHandler
	Location     *LocationHandler
	SystemConfig *SystemConfigHandler
	ScheduleRule *ScheduleRuleHandler
	Schedule     *ScheduleHandler
	Timetable    *TimetableHandler
	Export       *ExportHandler
}

// NewHandler 创建 Handler 聚合
func NewHandler(cfg *config.Config, svc *service.Service) *Handler {
	return &Handler{
		Auth:         NewAuthHandler(svc.Auth, &cfg.Auth.Cookie),
		User:         NewUserHandler(svc.User),
		Department:   NewDepartmentHandler(svc.Department),
		Semester:     NewSemesterHandler(svc.Semester),
		TimeSlot:     NewTimeSlotHandler(svc.TimeSlot),
		Location:     NewLocationHandler(svc.Location),
		SystemConfig: NewSystemConfigHandler(svc.SystemConfig),
		ScheduleRule: NewScheduleRuleHandler(svc.ScheduleRule),
		Schedule:     NewScheduleHandler(svc.Schedule),
		Timetable:    NewTimetableHandler(svc.Timetable),
		Export:       NewExportHandler(svc.Export),
	}
}
