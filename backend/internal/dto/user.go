package dto

// ── 用户模块 DTO ──

// UserListRequest 用户列表查询参数
type UserListRequest struct {
	PaginationRequest
	DepartmentID string `form:"department_id" binding:"omitempty,uuid"`
	Role         string `form:"role"          binding:"omitempty,oneof=admin leader member"`
	Keyword      string `form:"keyword"       binding:"omitempty,max=50"`
}

// UpdateUserRequest 更新用户信息请求
type UpdateUserRequest struct {
	Name         *string `json:"name"          binding:"omitempty,min=2,max=20"`
	Email        *string `json:"email"         binding:"omitempty,email"`
	DepartmentID *string `json:"department_id" binding:"omitempty,uuid"`
}

// AssignRoleRequest 分配角色请求
type AssignRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin leader member"`
}

// ResetPasswordResponse 重置密码响应
type ResetPasswordResponse struct {
	TempPassword string `json:"temp_password"`
}

// ImportUserResponse 批量导入用户响应
type ImportUserResponse struct {
	Total   int               `json:"total"`
	Success int               `json:"success"`
	Failed  int               `json:"failed"`
	Errors  []ImportUserError `json:"errors,omitempty"`
}

// ImportUserError 导入错误详情
type ImportUserError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// CreateUserRequest 手动新增用户请求
type CreateUserRequest struct {
	Name         string `json:"name"          binding:"required,min=2,max=20"`
	StudentID    string `json:"student_id"    binding:"required,min=1,max=20"`
	Email        string `json:"email"         binding:"required,email"`
	Role         string `json:"role"          binding:"required,oneof=admin leader member"`
	DepartmentID string `json:"department_id" binding:"required,uuid"`
}

// CreateUserResponse 新增用户响应
type CreateUserResponse struct {
	User         *UserResponse `json:"user"`
	TempPassword string        `json:"temp_password"`
}
