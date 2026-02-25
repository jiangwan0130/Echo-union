package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// DepartmentHandler 部门模块 HTTP 处理器
type DepartmentHandler struct {
	deptSvc service.DepartmentService
}

// NewDepartmentHandler 创建 DepartmentHandler
func NewDepartmentHandler(deptSvc service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{deptSvc: deptSvc}
}

// ListDepartments 获取部门列表
// GET /api/v1/departments
func (h *DepartmentHandler) ListDepartments(c *gin.Context) {
	var req dto.DepartmentListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	depts, err := h.deptSvc.List(c.Request.Context(), &req)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"list": depts})
}

// GetDepartment 获取部门详情
// GET /api/v1/departments/:id
func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "部门ID不能为空")
		return
	}

	dept, err := h.deptSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleDepartmentError(c, err)
		return
	}

	response.OK(c, dept)
}

// CreateDepartment 创建部门
// POST /api/v1/departments
func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var req dto.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	dept, err := h.deptSvc.Create(c.Request.Context(), &req, callerID.(string))
	if err != nil {
		h.handleDepartmentError(c, err)
		return
	}

	response.Created(c, dept)
}

// UpdateDepartment 更新部门
// PUT /api/v1/departments/:id
func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "部门ID不能为空")
		return
	}

	var req dto.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	dept, err := h.deptSvc.Update(c.Request.Context(), id, &req, callerID.(string))
	if err != nil {
		h.handleDepartmentError(c, err)
		return
	}

	response.OK(c, dept)
}

// DeleteDepartment 删除部门
// DELETE /api/v1/departments/:id
func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "部门ID不能为空")
		return
	}

	callerID, _ := c.Get("user_id")

	if err := h.deptSvc.Delete(c.Request.Context(), id, callerID.(string)); err != nil {
		h.handleDepartmentError(c, err)
		return
	}

	response.OK(c, nil)
}

// GetMembers 获取部门成员列表
// GET /api/v1/departments/:id/members
func (h *DepartmentHandler) GetMembers(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "部门ID不能为空")
		return
	}

	semesterID := c.Query("semester_id")

	members, err := h.deptSvc.GetMembers(c.Request.Context(), id, semesterID)
	if err != nil {
		h.handleDepartmentError(c, err)
		return
	}

	response.OK(c, gin.H{"list": members})
}

// SetDutyMembers 设置部门值班人员
// PUT /api/v1/departments/:id/duty-members
func (h *DepartmentHandler) SetDutyMembers(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "部门ID不能为空")
		return
	}

	var req dto.SetDutyMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	result, err := h.deptSvc.SetDutyMembers(c.Request.Context(), id, &req, callerID.(string))
	if err != nil {
		h.handleDepartmentError(c, err)
		return
	}

	response.OK(c, result)
}

// handleDepartmentError 统一处理部门模块业务错误
func (h *DepartmentHandler) handleDepartmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound):
		response.NotFound(c, 13001, "部门不存在")
	case errors.Is(err, service.ErrDepartmentNameExists):
		response.BadRequest(c, 13002, "部门名称已存在")
	case errors.Is(err, service.ErrDepartmentHasMembers):
		response.BadRequest(c, 13003, "部门下存在成员，无法删除")
	case errors.Is(err, service.ErrDepartmentInactive):
		response.BadRequest(c, 13004, "部门已停用")
	case errors.Is(err, service.ErrDutyMemberNotInDepartment):
		response.BadRequest(c, 13005, "指定用户不属于该部门")
	case errors.Is(err, service.ErrUserNotFound):
		response.NotFound(c, 13006, "指定用户不存在")
	case errors.Is(err, service.ErrSemesterNotFound):
		response.NotFound(c, 13007, "学期不存在")
	default:
		response.InternalError(c)
	}
}
