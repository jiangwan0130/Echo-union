package dto

// ── 部门模块 DTO ──

// CreateDepartmentRequest 创建部门请求
type CreateDepartmentRequest struct {
	Name        string `json:"name"        binding:"required,min=2,max=50"`
	Description string `json:"description" binding:"omitempty,max=200"`
}

// UpdateDepartmentRequest 更新部门请求
type UpdateDepartmentRequest struct {
	Name        *string `json:"name"        binding:"omitempty,min=2,max=50"`
	Description *string `json:"description" binding:"omitempty,max=200"`
	IsActive    *bool   `json:"is_active"`
}

// DepartmentListRequest 部门列表查询参数
type DepartmentListRequest struct {
	IncludeInactive bool `form:"include_inactive"`
}

// DepartmentDetailResponse 部门详细信息响应
type DepartmentDetailResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
	MemberCount int64  `json:"member_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ── 部门成员管理 DTO ──

// DepartmentMemberResponse 部门成员响应
type DepartmentMemberResponse struct {
	UserID          string `json:"user_id"`
	Name            string `json:"name"`
	StudentID       string `json:"student_id"`
	Email           string `json:"email"`
	Role            string `json:"role"`
	DutyRequired    bool   `json:"duty_required"`
	TimetableStatus string `json:"timetable_status"`
}

// SetDutyMembersRequest 设置值班人员请求
type SetDutyMembersRequest struct {
	SemesterID string   `json:"semester_id" binding:"required,uuid"`
	UserIDs    []string `json:"user_ids"    binding:"required,min=1,dive,uuid"`
}

// SetDutyMembersResponse 设置值班人员响应
type SetDutyMembersResponse struct {
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	SemesterID     string `json:"semester_id"`
	TotalSet       int    `json:"total_set"`
}
