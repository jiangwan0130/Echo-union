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
	Phase         string `json:"phase"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ── 阶段推进 DTO ──

// AdvancePhaseRequest 阶段推进请求
type AdvancePhaseRequest struct {
	TargetPhase string `json:"target_phase" binding:"required,oneof=configuring collecting scheduling published"`
}

// PhaseCheckResponse 阶段完成条件检查响应
type PhaseCheckResponse struct {
	CurrentPhase string           `json:"current_phase"`
	CanAdvance   bool             `json:"can_advance"`
	Checks       []PhaseCheckItem `json:"checks"`
}

// PhaseCheckItem 单项检查结果
type PhaseCheckItem struct {
	Label   string `json:"label"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// DutyMembersRequest 批量设置值班人员请求
type DutyMembersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

// DutyMemberItem 值班人员信息
type DutyMemberItem struct {
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	StudentID      string `json:"student_id"`
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	DutyRequired   bool   `json:"duty_required"`
}

// PendingTodoItem 待办事项
type PendingTodoItem struct {
	Type    string `json:"type"` // submit_timetable | schedule_published | waiting_schedule
	Title   string `json:"title"`
	Message string `json:"message"`
}
