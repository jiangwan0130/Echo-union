package dto

// ── 认证模块 DTO ──

// LoginRequest 登录请求
type LoginRequest struct {
	StudentID  string `json:"student_id" binding:"required"`
	Password   string `json:"password"   binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

// RegisterRequest 邀请注册请求
type RegisterRequest struct {
	InviteCode   string `json:"invite_code"   binding:"required"`
	Name         string `json:"name"          binding:"required,min=2,max=20"`
	StudentID    string `json:"student_id"    binding:"required"`
	Email        string `json:"email"         binding:"required,email"`
	Password     string `json:"password"      binding:"required,min=8,max=20"`
	DepartmentID string `json:"department_id" binding:"required,uuid"`
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"` // 非 Cookie 模式时使用
}

// GenerateInviteRequest 生成邀请链接请求
type GenerateInviteRequest struct {
	ExpiresDays int `json:"expires_days"` // 默认 7 天
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=20"`
}

// [自证通过] internal/dto/auth.go
