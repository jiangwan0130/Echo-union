package handler

import (
	"errors"
	"strings"

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

// CreateUser 手动新增用户
// POST /api/v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	result, err := h.userSvc.CreateUser(c.Request.Context(), &req, callerID)
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	response.Created(c, result)
}

// GetCurrentUser 获取当前用户信息
// GET /api/v1/users/me
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	user, err := h.userSvc.GetByID(c.Request.Context(), userID)
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	response.OK(c, user)
}

// GetUser 获取指定用户详情
// GET /api/v1/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "用户ID不能为空")
		return
	}

	user, err := h.userSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	response.OK(c, user)
}

// ListUsers 用户列表（管理员/部门负责人）
// GET /api/v1/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req dto.UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerRole, ok := MustGetRole(c)
	if !ok {
		return
	}
	callerDeptID, ok := MustGetDepartmentID(c)
	if !ok {
		return
	}

	users, total, err := h.userSvc.List(
		c.Request.Context(),
		&req,
		callerRole,
		callerDeptID,
	)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OKPage(c, users, total, req.GetPage(), req.GetPageSize())
}

// UpdateUser 更新用户信息
// PUT /api/v1/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "用户ID不能为空")
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}
	callerRole, ok := MustGetRole(c)
	if !ok {
		return
	}

	user, err := h.userSvc.Update(c.Request.Context(), id, &req, callerID, callerRole)
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	response.OK(c, user)
}

// DeleteUser 删除用户（软删除）
// DELETE /api/v1/users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "用户ID不能为空")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	if err := h.userSvc.Delete(c.Request.Context(), id, callerID); err != nil {
		h.handleUserError(c, err)
		return
	}

	response.OK(c, nil)
}

// AssignRole 分配角色
// PUT /api/v1/users/:id/role
func (h *UserHandler) AssignRole(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "用户ID不能为空")
		return
	}

	var req dto.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	if err := h.userSvc.AssignRole(c.Request.Context(), id, &req, callerID); err != nil {
		h.handleUserError(c, err)
		return
	}

	response.OK(c, nil)
}

// ResetPassword 管理员重置用户密码
// POST /api/v1/users/:id/reset-password
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "用户ID不能为空")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	result, err := h.userSvc.ResetPassword(c.Request.Context(), id, callerID)
	if err != nil {
		h.handleUserError(c, err)
		return
	}

	response.OK(c, result)
}

// ImportUsers 批量导入用户（Excel）
// POST /api/v1/users/import
func (h *UserHandler) ImportUsers(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, 10001, "请上传Excel文件")
		return
	}

	// 校验文件扩展名
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".xlsx") {
		response.BadRequest(c, 10001, "仅支持 .xlsx 格式")
		return
	}

	// 限制文件大小 (5MB)
	if file.Size > 5*1024*1024 {
		response.BadRequest(c, 10001, "文件大小不能超过5MB")
		return
	}

	src, err := file.Open()
	if err != nil {
		response.InternalError(c)
		return
	}
	defer src.Close()

	// 委托 Service 层解析 Excel
	rows, err := h.userSvc.ParseImportFile(src)
	if err != nil {
		response.BadRequest(c, 10001, err.Error())
		return
	}

	result, err := h.userSvc.ImportUsers(c.Request.Context(), rows)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, result)
}

// ── 内部辅助方法 ──

// handleUserError 统一处理用户模块业务错误
func (h *UserHandler) handleUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		response.NotFound(c, 12001, "用户不存在")
	case errors.Is(err, service.ErrUserSelfRoleChange):
		response.BadRequest(c, 12002, "无法修改自己的角色")
	case errors.Is(err, service.ErrUserSelfDelete):
		response.BadRequest(c, 12003, "无法删除自己")
	case errors.Is(err, service.ErrEmailExists):
		response.BadRequest(c, 12004, "邮箱已被使用")
	case errors.Is(err, service.ErrDepartmentNotFound):
		response.BadRequest(c, 12005, "部门不存在")
	case errors.Is(err, service.ErrStudentIDExists):
		response.BadRequest(c, 12006, "学号已被使用")
	case errors.Is(err, service.ErrNoPermission):
		response.Forbidden(c, 10003, "无权操作")
	default:
		response.InternalError(c)
	}
}
