package dto

// ── 导出模块 DTO ──

// ExportScheduleRequest 导出排班表请求
type ExportScheduleRequest struct {
	SemesterID string `form:"semester_id" binding:"required,uuid"`
}
