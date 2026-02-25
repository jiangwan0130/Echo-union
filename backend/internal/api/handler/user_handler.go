package handler

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

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

	callerRole, _ := c.Get("role")
	callerDeptID, _ := c.Get("department_id")

	users, total, err := h.userSvc.List(
		c.Request.Context(),
		&req,
		callerRole.(string),
		callerDeptID.(string),
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

	callerID, _ := c.Get("user_id")
	callerRole, _ := c.Get("role")

	user, err := h.userSvc.Update(c.Request.Context(), id, &req, callerID.(string), callerRole.(string))
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

	callerID, _ := c.Get("user_id")

	if err := h.userSvc.Delete(c.Request.Context(), id, callerID.(string)); err != nil {
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

	callerID, _ := c.Get("user_id")

	if err := h.userSvc.AssignRole(c.Request.Context(), id, &req, callerID.(string)); err != nil {
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

	callerID, _ := c.Get("user_id")

	result, err := h.userSvc.ResetPassword(c.Request.Context(), id, callerID.(string))
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

	f, err := excelize.OpenReader(src)
	if err != nil {
		response.BadRequest(c, 10001, "无法解析Excel文件")
		return
	}
	defer f.Close()

	// 读取第一个工作表
	sheetName := f.GetSheetName(0)
	excelRows, err := f.GetRows(sheetName)
	if err != nil {
		response.BadRequest(c, 10001, "读取工作表失败")
		return
	}

	if len(excelRows) < 2 {
		response.BadRequest(c, 10001, "Excel文件无数据行（第一行为表头）")
		return
	}

	// 解析表头（支持灵活列序）
	header := excelRows[0]
	colIndex := parseHeaderIndex(header)

	if colIndex["name"] < 0 || colIndex["student_id"] < 0 || colIndex["email"] < 0 || colIndex["department"] < 0 {
		response.BadRequest(c, 10001, "Excel表头缺少必要列（姓名/学号/邮箱/部门）")
		return
	}

	// 解析数据行
	var rows []service.ImportUserRow
	for i := 1; i < len(excelRows); i++ {
		row := excelRows[i]
		item := service.ImportUserRow{Row: i + 1} // Excel行号从1开始，+1因为表头

		if idx := colIndex["name"]; idx < len(row) {
			item.Name = strings.TrimSpace(row[idx])
		}
		if idx := colIndex["student_id"]; idx < len(row) {
			item.StudentID = strings.TrimSpace(row[idx])
		}
		if idx := colIndex["email"]; idx < len(row) {
			item.Email = strings.TrimSpace(row[idx])
		}
		if idx := colIndex["department"]; idx < len(row) {
			item.DepartmentName = strings.TrimSpace(row[idx])
		}

		rows = append(rows, item)
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
	case errors.Is(err, service.ErrNoPermission):
		response.Forbidden(c, 10003, "无权操作")
	default:
		response.InternalError(c)
	}
}

// parseHeaderIndex 解析 Excel 表头，返回列名 -> 列索引映射
// 支持的列名：姓名/name、学号/student_id、邮箱/email、部门/department
func parseHeaderIndex(header []string) map[string]int {
	idx := map[string]int{
		"name":       -1,
		"student_id": -1,
		"email":      -1,
		"department": -1,
	}

	for i, h := range header {
		lower := strings.ToLower(strings.TrimSpace(h))
		switch {
		case lower == "姓名" || lower == "name":
			idx["name"] = i
		case lower == "学号" || lower == "student_id":
			idx["student_id"] = i
		case lower == "邮箱" || lower == "email":
			idx["email"] = i
		case lower == "部门" || lower == "department":
			idx["department"] = i
		}
	}

	return idx
}
