package dto

// ── 认证模块 DTO ──

// LoginRequest 登录请求
type LoginRequest struct {
	StudentID  string `json:"student_id" binding:"required"`
	Password   string `json:"password"   binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"` // 非 Cookie 模式时使用
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=20"`
}

// [自证通过] internal/dto/auth.go
