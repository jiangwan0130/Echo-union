package dto

// ── 学期模块 DTO ──

// CreateSemesterRequest 创建学期请求
type CreateSemesterRequest struct {
	Name          string `json:"name"            binding:"required,min=2,max=100"`
	StartDate     string `json:"start_date"      binding:"required"` // "2026-09-01"
	EndDate       string `json:"end_date"        binding:"required"` // "2027-01-15"
	FirstWeekType string `json:"first_week_type" binding:"required,oneof=odd even"`
}

// UpdateSemesterRequest 更新学期请求
type UpdateSemesterRequest struct {
	Name          *string `json:"name"            binding:"omitempty,min=2,max=100"`
	StartDate     *string `json:"start_date"`
	EndDate       *string `json:"end_date"`
	FirstWeekType *string `json:"first_week_type" binding:"omitempty,oneof=odd even"`
	Status        *string `json:"status"          binding:"omitempty,oneof=active archived"`
}

// SemesterResponse 学期信息响应
type SemesterResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	FirstWeekType string `json:"first_week_type"`
	IsActive      bool   `json:"is_active"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}
