package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"echo-union/backend/config"
	"echo-union/backend/internal/api/handler"
	"echo-union/backend/internal/api/middleware"
	"echo-union/backend/pkg/jwt"
	"echo-union/backend/pkg/redis"
)

// Setup 初始化并返回 Gin 路由引擎
func Setup(cfg *config.Config, h *handler.Handler, jwtMgr *jwt.Manager, rdb *redis.Client, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// ── 全局中间件 ──
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.Server.CORS.AllowOrigins))

	// ── 健康检查 ──
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ── API v1 ──
	v1 := r.Group("/api/v1")
	{
		// 认证模块（无需认证）
		auth := v1.Group("/auth")
		{
			auth.POST("/login", h.Auth.Login)
			auth.POST("/register", h.Auth.Register)
			auth.POST("/refresh", h.Auth.RefreshToken)
			auth.GET("/invite/:code", h.Auth.ValidateInvite)
		}

		// 需要认证的路由
		authorized := v1.Group("")
		authorized.Use(middleware.JWTAuth(jwtMgr, rdb))
		{
			// 认证模块（需要认证）
			authorized.POST("/auth/logout", h.Auth.Logout)
			authorized.GET("/auth/me", h.Auth.GetCurrentUser)
			authorized.PUT("/auth/password", h.Auth.ChangePassword)
			authorized.POST("/auth/invite", middleware.RoleAuth("admin", "leader"), h.Auth.GenerateInvite)

			// 用户模块
			users := authorized.Group("/users")
			{
				users.GET("/me", h.User.GetCurrentUser)
				users.GET("", middleware.RoleAuth("admin", "leader"), h.User.ListUsers)
				users.GET("/:id", middleware.RoleAuth("admin", "leader"), h.User.GetUser)
				users.PUT("/:id", h.User.UpdateUser) // admin 或本人（Service 层鉴权）
				users.DELETE("/:id", middleware.RoleAuth("admin"), h.User.DeleteUser)
				users.PUT("/:id/role", middleware.RoleAuth("admin"), h.User.AssignRole)
				users.POST("/:id/reset-password", middleware.RoleAuth("admin"), h.User.ResetPassword)
				users.POST("/import", middleware.RoleAuth("admin"), h.User.ImportUsers)
			}

			// 部门模块
			departments := authorized.Group("/departments")
			{
				departments.GET("", h.Department.ListDepartments)
				departments.GET("/:id", h.Department.GetDepartment)
				departments.POST("", middleware.RoleAuth("admin"), h.Department.CreateDepartment)
				departments.PUT("/:id", middleware.RoleAuth("admin"), h.Department.UpdateDepartment)
				departments.DELETE("/:id", middleware.RoleAuth("admin"), h.Department.DeleteDepartment)
				departments.GET("/:id/members", middleware.RoleAuth("admin", "leader"), h.Department.GetMembers)
				departments.PUT("/:id/duty-members", middleware.RoleAuth("admin", "leader"), h.Department.SetDutyMembers)
			}

			// 学期模块
			semesters := authorized.Group("/semesters")
			{
				semesters.GET("", h.Semester.ListSemesters)
				semesters.GET("/current", h.Semester.GetCurrentSemester)
				semesters.GET("/:id", h.Semester.GetSemester)
				semesters.POST("", middleware.RoleAuth("admin"), h.Semester.CreateSemester)
				semesters.PUT("/:id", middleware.RoleAuth("admin"), h.Semester.UpdateSemester)
				semesters.PUT("/:id/activate", middleware.RoleAuth("admin"), h.Semester.ActivateSemester)
				semesters.DELETE("/:id", middleware.RoleAuth("admin"), h.Semester.DeleteSemester)
			}

			// 时间段模块
			timeSlots := authorized.Group("/time-slots")
			{
				timeSlots.GET("", h.TimeSlot.ListTimeSlots)
				timeSlots.GET("/:id", h.TimeSlot.GetTimeSlot)
				timeSlots.POST("", middleware.RoleAuth("admin"), h.TimeSlot.CreateTimeSlot)
				timeSlots.PUT("/:id", middleware.RoleAuth("admin"), h.TimeSlot.UpdateTimeSlot)
				timeSlots.DELETE("/:id", middleware.RoleAuth("admin"), h.TimeSlot.DeleteTimeSlot)
			}

			// 地点模块
			locations := authorized.Group("/locations")
			{
				locations.GET("", h.Location.ListLocations)
				locations.GET("/:id", h.Location.GetLocation)
				locations.POST("", middleware.RoleAuth("admin"), h.Location.CreateLocation)
				locations.PUT("/:id", middleware.RoleAuth("admin"), h.Location.UpdateLocation)
				locations.DELETE("/:id", middleware.RoleAuth("admin"), h.Location.DeleteLocation)
			}

			// 系统配置模块
			systemConfig := authorized.Group("/system-config")
			{
				systemConfig.GET("", h.SystemConfig.GetConfig)
				systemConfig.PUT("", middleware.RoleAuth("admin"), h.SystemConfig.UpdateConfig)
			}

			// 排班规则模块
			scheduleRules := authorized.Group("/schedule-rules")
			{
				scheduleRules.GET("", h.ScheduleRule.ListRules)
				scheduleRules.GET("/:id", h.ScheduleRule.GetRule)
				scheduleRules.PUT("/:id", middleware.RoleAuth("admin"), h.ScheduleRule.UpdateRule)
			}

			// 时间表模块
			timetables := authorized.Group("/timetables")
			{
				timetables.POST("/import", h.Timetable.ImportICS)
				timetables.GET("/me", h.Timetable.GetMyTimetable)
				timetables.POST("/unavailable", h.Timetable.CreateUnavailableTime)
				timetables.PUT("/unavailable/:id", h.Timetable.UpdateUnavailableTime)
				timetables.DELETE("/unavailable/:id", h.Timetable.DeleteUnavailableTime)
				timetables.POST("/submit", h.Timetable.SubmitTimetable)
				timetables.GET("/progress", middleware.RoleAuth("admin"), h.Timetable.GetProgress)
				timetables.GET("/progress/department/:id", middleware.RoleAuth("admin", "leader"), h.Timetable.GetDepartmentProgress)
			}

			// 排班模块
			schedules := authorized.Group("/schedules")
			{
				schedules.POST("/auto", middleware.RoleAuth("admin"), h.Schedule.AutoSchedule)
				schedules.GET("", h.Schedule.GetSchedule)
				schedules.GET("/my", h.Schedule.GetMySchedule)
				schedules.PUT("/items/:id", middleware.RoleAuth("admin"), h.Schedule.UpdateItem)
				schedules.POST("/items/:id/validate", middleware.RoleAuth("admin"), h.Schedule.ValidateCandidate)
				schedules.GET("/items/:id/candidates", middleware.RoleAuth("admin"), h.Schedule.GetCandidates)
				schedules.POST("/publish", middleware.RoleAuth("admin"), h.Schedule.Publish)
				schedules.PUT("/published/items/:id", middleware.RoleAuth("admin"), h.Schedule.UpdatePublishedItem)
				schedules.GET("/change-logs", middleware.RoleAuth("admin"), h.Schedule.ListChangeLogs)
				schedules.GET("/:id/scope-check", middleware.RoleAuth("admin"), h.Schedule.CheckScope)
			}

			// 导出模块（一期：排班表导出；签到统计导出归入二期）
			export := authorized.Group("/export")
			{
				export.GET("/schedule", middleware.RoleAuth("admin", "leader"), h.Export.ExportSchedule)
			}
		}
	}

	return r
}
