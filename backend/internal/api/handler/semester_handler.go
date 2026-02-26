package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// SemesterHandler 学期模块 HTTP 处理器
type SemesterHandler struct {
	semesterSvc service.SemesterService
}

// NewSemesterHandler 创建 SemesterHandler
func NewSemesterHandler(semesterSvc service.SemesterService) *SemesterHandler {
	return &SemesterHandler{semesterSvc: semesterSvc}
}

// ListSemesters 获取学期列表
// GET /api/v1/semesters
func (h *SemesterHandler) ListSemesters(c *gin.Context) {
	semesters, err := h.semesterSvc.List(c.Request.Context())
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"list": semesters})
}

// GetSemester 获取学期详情
// GET /api/v1/semesters/:id
func (h *SemesterHandler) GetSemester(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "学期ID不能为空")
		return
	}

	semester, err := h.semesterSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleSemesterError(c, err)
		return
	}

	response.OK(c, semester)
}

// GetCurrentSemester 获取当前学期
// GET /api/v1/semesters/current
func (h *SemesterHandler) GetCurrentSemester(c *gin.Context) {
	semester, err := h.semesterSvc.GetCurrent(c.Request.Context())
	if err != nil {
		h.handleSemesterError(c, err)
		return
	}

	response.OK(c, semester)
}

// CreateSemester 创建学期
// POST /api/v1/semesters
func (h *SemesterHandler) CreateSemester(c *gin.Context) {
	var req dto.CreateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	semester, err := h.semesterSvc.Create(c.Request.Context(), &req, callerID)
	if err != nil {
		h.handleSemesterError(c, err)
		return
	}

	response.Created(c, semester)
}

// UpdateSemester 更新学期
// PUT /api/v1/semesters/:id
func (h *SemesterHandler) UpdateSemester(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "学期ID不能为空")
		return
	}

	var req dto.UpdateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	semester, err := h.semesterSvc.Update(c.Request.Context(), id, &req, callerID)
	if err != nil {
		h.handleSemesterError(c, err)
		return
	}

	response.OK(c, semester)
}

// ActivateSemester 激活学期（设为当前学期）
// PUT /api/v1/semesters/:id/activate
func (h *SemesterHandler) ActivateSemester(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "学期ID不能为空")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	if err := h.semesterSvc.Activate(c.Request.Context(), id, callerID); err != nil {
		h.handleSemesterError(c, err)
		return
	}

	response.OK(c, nil)
}

// DeleteSemester 删除学期
// DELETE /api/v1/semesters/:id
func (h *SemesterHandler) DeleteSemester(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "学期ID不能为空")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	if err := h.semesterSvc.Delete(c.Request.Context(), id, callerID); err != nil {
		h.handleSemesterError(c, err)
		return
	}

	response.OK(c, nil)
}

// handleSemesterError 统一处理学期模块业务错误
func (h *SemesterHandler) handleSemesterError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSemesterNotFound):
		response.NotFound(c, 14001, "学期不存在")
	case errors.Is(err, service.ErrSemesterDateInvalid):
		response.BadRequest(c, 14002, "学期日期无效")
	case errors.Is(err, service.ErrSemesterDateOverlap):
		response.BadRequest(c, 14003, "学期日期与已有学期重叠")
	default:
		response.InternalError(c)
	}
}
