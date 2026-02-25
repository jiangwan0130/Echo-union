package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// ScheduleHandler 排班模块 HTTP 处理器
type ScheduleHandler struct {
	scheduleSvc service.ScheduleService
}

// NewScheduleHandler 创建 ScheduleHandler
func NewScheduleHandler(scheduleSvc service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleSvc: scheduleSvc}
}

// AutoSchedule 执行自动排班
// POST /api/v1/schedules/auto
func (h *ScheduleHandler) AutoSchedule(c *gin.Context) {
	var req dto.AutoScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 13001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	result, err := h.scheduleSvc.AutoSchedule(c.Request.Context(), &req, callerID.(string))
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, result)
}

// GetSchedule 获取排班表
// GET /api/v1/schedules
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	semesterID := c.Query("semester_id")
	if semesterID == "" {
		response.BadRequest(c, 13001, "semester_id不能为空")
		return
	}

	schedule, err := h.scheduleSvc.GetSchedule(c.Request.Context(), semesterID)
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, schedule)
}

// GetMySchedule 获取我的排班
// GET /api/v1/schedules/my
func (h *ScheduleHandler) GetMySchedule(c *gin.Context) {
	semesterID := c.Query("semester_id")
	if semesterID == "" {
		response.BadRequest(c, 13001, "semester_id不能为空")
		return
	}

	userID, _ := c.Get("user_id")

	items, err := h.scheduleSvc.GetMySchedule(c.Request.Context(), semesterID, userID.(string))
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, gin.H{"list": items})
}

// UpdateItem 手动调整排班项
// PUT /api/v1/schedules/items/:id
func (h *ScheduleHandler) UpdateItem(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 13001, "排班项ID不能为空")
		return
	}

	var req dto.UpdateScheduleItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 13001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	item, err := h.scheduleSvc.UpdateItem(c.Request.Context(), id, &req, callerID.(string))
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, item)
}

// ValidateCandidate 校验候选人是否可排
// POST /api/v1/schedules/items/:id/validate
func (h *ScheduleHandler) ValidateCandidate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 13001, "排班项ID不能为空")
		return
	}

	var req dto.ValidateCandidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 13001, "参数校验失败")
		return
	}

	result, err := h.scheduleSvc.ValidateCandidate(c.Request.Context(), id, &req)
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, result)
}

// GetCandidates 获取时段可用候选人
// GET /api/v1/schedules/items/:id/candidates
func (h *ScheduleHandler) GetCandidates(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 13001, "排班项ID不能为空")
		return
	}

	candidates, err := h.scheduleSvc.GetCandidates(c.Request.Context(), id)
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, gin.H{"list": candidates})
}

// Publish 发布排班表
// POST /api/v1/schedules/publish
func (h *ScheduleHandler) Publish(c *gin.Context) {
	var req dto.PublishScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 13001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	schedule, err := h.scheduleSvc.Publish(c.Request.Context(), &req, callerID.(string))
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, schedule)
}

// UpdatePublishedItem 发布后修改排班项
// PUT /api/v1/schedules/published/items/:id
func (h *ScheduleHandler) UpdatePublishedItem(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 13001, "排班项ID不能为空")
		return
	}

	var req dto.UpdatePublishedItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 13001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	item, err := h.scheduleSvc.UpdatePublishedItem(c.Request.Context(), id, &req, callerID.(string))
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, item)
}

// ListChangeLogs 获取变更日志
// GET /api/v1/schedules/change-logs
func (h *ScheduleHandler) ListChangeLogs(c *gin.Context) {
	var req dto.ScheduleChangeLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, 13001, "参数校验失败")
		return
	}

	logs, total, err := h.scheduleSvc.ListChangeLogs(c.Request.Context(), &req)
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OKPage(c, logs, total, req.GetPage(), req.GetPageSize())
}

// CheckScope 范围检测
// GET /api/v1/schedules/:id/scope-check
func (h *ScheduleHandler) CheckScope(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 13001, "排班表ID不能为空")
		return
	}

	result, err := h.scheduleSvc.CheckScope(c.Request.Context(), id)
	if err != nil {
		h.handleScheduleError(c, err)
		return
	}

	response.OK(c, result)
}

// handleScheduleError 统一处理排班模块业务错误
func (h *ScheduleHandler) handleScheduleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrScheduleNotFound):
		response.NotFound(c, 13101, "排班表不存在")
	case errors.Is(err, service.ErrScheduleItemNotFound):
		response.NotFound(c, 13102, "排班项不存在")
	case errors.Is(err, service.ErrScheduleAlreadyExists):
		response.BadRequest(c, 13103, "该学期已存在排班表")
	case errors.Is(err, service.ErrScheduleNotDraft):
		response.BadRequest(c, 13104, "排班表非草稿状态，不可执行此操作")
	case errors.Is(err, service.ErrScheduleNotPublished):
		response.BadRequest(c, 13105, "排班表非已发布状态")
	case errors.Is(err, service.ErrScheduleCannotPublish):
		response.BadRequest(c, 13106, "排班表不可发布")
	case errors.Is(err, service.ErrSubmissionRateIncomplete):
		response.BadRequest(c, 13107, "课表提交率未达100%，请确保所有需值班成员已提交课表")
	case errors.Is(err, service.ErrNoEligibleMembers):
		response.BadRequest(c, 13108, "无符合条件的排班候选人")
	case errors.Is(err, service.ErrNoActiveTimeSlots):
		response.BadRequest(c, 13109, "无可用时间段")
	case errors.Is(err, service.ErrCandidateNotAvailable):
		response.BadRequest(c, 13110, "候选人在该时段不可用")
	case errors.Is(err, service.ErrSemesterNotFound):
		response.NotFound(c, 13111, "学期不存在")
	default:
		response.InternalError(c)
	}
}
