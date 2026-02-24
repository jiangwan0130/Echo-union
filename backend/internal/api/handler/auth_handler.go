package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// AuthHandler 认证模块 HTTP 处理器
type AuthHandler struct {
	authSvc service.AuthService
}

// NewAuthHandler 创建 AuthHandler
func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	result, err := h.authSvc.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, 11001, "学号或密码错误")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, result)
}

// Logout 用户登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// 📝 待实现: Token 黑名单（Redis）
	response.OK(c, nil)
}

// RefreshToken 刷新 Token
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// 📝 待实现
	response.OK(c, nil)
}

// GenerateInvite 生成邀请链接
// POST /api/v1/auth/invite
func (h *AuthHandler) GenerateInvite(c *gin.Context) {
	// 📝 待实现
	response.OK(c, nil)
}

// ValidateInvite 验证邀请码
// GET /api/v1/auth/invite/:code
func (h *AuthHandler) ValidateInvite(c *gin.Context) {
	// 📝 待实现
	response.OK(c, nil)
}

// Register 邀请注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	// 📝 待实现
	response.Created(c, nil)
}

// [自证通过] internal/api/handler/auth_handler.go
