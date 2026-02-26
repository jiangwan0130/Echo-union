package dto

// ── 认证模块响应 ──

// TokenResponse Token 对响应
type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token,omitempty"` // Cookie 模式下可不返回
	ExpiresIn    int          `json:"expires_in"`              // Access Token 有效期（秒）
	User         UserResponse `json:"user"`
}

// InviteResponse 邀请码响应
type InviteResponse struct {
	InviteCode string `json:"invite_code"`
	InviteURL  string `json:"invite_url"`
	ExpiresAt  string `json:"expires_at"`
}

// InviteValidateResponse 邀请码验证响应
type InviteValidateResponse struct {
	Valid     bool   `json:"valid"`
	ExpiresAt string `json:"expires_at"`
}

// RegisterResponse 注册成功响应
type RegisterResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ── 用户模块响应 ──

// UserResponse 用户信息响应（脱敏）
type UserResponse struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Email              string              `json:"email"`
	StudentID          string              `json:"student_id"`
	Role               string              `json:"role"`
	Department         *DepartmentResponse `json:"department,omitempty"`
	MustChangePassword bool                `json:"must_change_password"`
}

// UserDetailResponse 用户详细信息（GET /auth/me）
type UserDetailResponse struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Email              string              `json:"email"`
	StudentID          string              `json:"student_id"`
	Role               string              `json:"role"`
	Department         *DepartmentResponse `json:"department,omitempty"`
	MustChangePassword bool                `json:"must_change_password"`
	CreatedAt          string              `json:"created_at"`
}

// DepartmentResponse 部门简要信息
type DepartmentResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ── 分页请求 ──

// PaginationRequest 通用分页参数
type PaginationRequest struct {
	Page     int `form:"page"      binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// GetPage 获取页码（含默认值）
func (p *PaginationRequest) GetPage() int {
	if p.Page <= 0 {
		return 1
	}
	return p.Page
}

// GetPageSize 获取每页数量（含默认值）
func (p *PaginationRequest) GetPageSize() int {
	if p.PageSize <= 0 {
		return 20
	}
	return p.PageSize
}

// GetOffset 计算偏移量
func (p *PaginationRequest) GetOffset() int {
	return (p.GetPage() - 1) * p.GetPageSize()
}

// [自证通过] internal/dto/response.go
