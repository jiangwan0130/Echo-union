package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// TimeSlotHandler 时间段模块 HTTP 处理器
type TimeSlotHandler struct {
	timeSlotSvc service.TimeSlotService
}

// NewTimeSlotHandler 创建 TimeSlotHandler
func NewTimeSlotHandler(timeSlotSvc service.TimeSlotService) *TimeSlotHandler {
	return &TimeSlotHandler{timeSlotSvc: timeSlotSvc}
}

// ListTimeSlots 获取时间段列表
// GET /api/v1/time-slots
func (h *TimeSlotHandler) ListTimeSlots(c *gin.Context) {
	var req dto.TimeSlotListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	slots, err := h.timeSlotSvc.List(c.Request.Context(), &req)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"list": slots})
}

// GetTimeSlot 获取时间段详情
// GET /api/v1/time-slots/:id
func (h *TimeSlotHandler) GetTimeSlot(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "时间段ID不能为空")
		return
	}

	slot, err := h.timeSlotSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleTimeSlotError(c, err)
		return
	}

	response.OK(c, slot)
}

// CreateTimeSlot 创建时间段
// POST /api/v1/time-slots
func (h *TimeSlotHandler) CreateTimeSlot(c *gin.Context) {
	var req dto.CreateTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	slot, err := h.timeSlotSvc.Create(c.Request.Context(), &req, callerID)
	if err != nil {
		h.handleTimeSlotError(c, err)
		return
	}

	response.Created(c, slot)
}

// UpdateTimeSlot 更新时间段
// PUT /api/v1/time-slots/:id
func (h *TimeSlotHandler) UpdateTimeSlot(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "时间段ID不能为空")
		return
	}

	var req dto.UpdateTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	slot, err := h.timeSlotSvc.Update(c.Request.Context(), id, &req, callerID)
	if err != nil {
		h.handleTimeSlotError(c, err)
		return
	}

	response.OK(c, slot)
}

// DeleteTimeSlot 删除时间段
// DELETE /api/v1/time-slots/:id
func (h *TimeSlotHandler) DeleteTimeSlot(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "时间段ID不能为空")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	if err := h.timeSlotSvc.Delete(c.Request.Context(), id, callerID); err != nil {
		h.handleTimeSlotError(c, err)
		return
	}

	response.OK(c, nil)
}

// handleTimeSlotError 统一处理时间段模块业务错误
func (h *TimeSlotHandler) handleTimeSlotError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrTimeSlotNotFound):
		response.NotFound(c, 15001, "时间段不存在")
	case errors.Is(err, service.ErrSemesterNotFound):
		response.BadRequest(c, 15002, "关联的学期不存在")
	default:
		response.InternalError(c)
	}
}
