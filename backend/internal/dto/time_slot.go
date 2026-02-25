package dto

// ── 时间段模块 DTO ──

// CreateTimeSlotRequest 创建时间段请求
type CreateTimeSlotRequest struct {
	Name       string  `json:"name"        binding:"required,min=2,max=50"`
	SemesterID *string `json:"semester_id" binding:"omitempty,uuid"`
	StartTime  string  `json:"start_time"  binding:"required"` // "08:10"
	EndTime    string  `json:"end_time"    binding:"required"` // "10:05"
	DayOfWeek  int     `json:"day_of_week" binding:"required,min=1,max=5"`
}

// UpdateTimeSlotRequest 更新时间段请求
type UpdateTimeSlotRequest struct {
	Name      *string `json:"name"       binding:"omitempty,min=2,max=50"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	DayOfWeek *int    `json:"day_of_week" binding:"omitempty,min=1,max=5"`
	IsActive  *bool   `json:"is_active"`
}

// TimeSlotListRequest 时间段列表查询参数
type TimeSlotListRequest struct {
	SemesterID string `form:"semester_id" binding:"omitempty,uuid"`
	DayOfWeek  *int   `form:"day_of_week" binding:"omitempty,min=1,max=5"`
}

// TimeSlotResponse 时间段信息响应
type TimeSlotResponse struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	SemesterID *string        `json:"semester_id,omitempty"`
	Semester   *SemesterBrief `json:"semester,omitempty"`
	StartTime  string         `json:"start_time"`
	EndTime    string         `json:"end_time"`
	DayOfWeek  int            `json:"day_of_week"`
	IsActive   bool           `json:"is_active"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

// SemesterBrief 学期简要信息（嵌入时间段响应）
type SemesterBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
