package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// UserHandler 用户模块 HTTP 处理器
type UserHandler struct {
	userSvc service.UserService
}

// NewUserHandler 创建 UserHandler
func NewUserHandler(userSvc service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// GetCurrentUser 获取当前用户信息
// GET /api/v1/users/me
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, 10002, "未认证")
		return
	}

	user, err := h.userSvc.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, 20001, "用户不存在")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, user)
}

// ListUsers 用户列表（管理员）
// GET /api/v1/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	var page dto.PaginationRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	users, total, err := h.userSvc.List(c.Request.Context(), &page)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OKPage(c, users, total, page.GetPage(), page.GetPageSize())
}

// [自证通过] internal/api/handler/user_handler.go
