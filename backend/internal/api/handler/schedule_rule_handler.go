package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// ScheduleRuleHandler 排班规则模块 HTTP 处理器
type ScheduleRuleHandler struct {
	ruleSvc service.ScheduleRuleService
}

// NewScheduleRuleHandler 创建 ScheduleRuleHandler
func NewScheduleRuleHandler(ruleSvc service.ScheduleRuleService) *ScheduleRuleHandler {
	return &ScheduleRuleHandler{ruleSvc: ruleSvc}
}

// ListRules 获取排班规则列表
// GET /api/v1/schedule-rules
func (h *ScheduleRuleHandler) ListRules(c *gin.Context) {
	rules, err := h.ruleSvc.List(c.Request.Context())
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"list": rules})
}

// GetRule 获取排班规则详情
// GET /api/v1/schedule-rules/:id
func (h *ScheduleRuleHandler) GetRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "规则ID不能为空")
		return
	}

	rule, err := h.ruleSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleRuleError(c, err)
		return
	}

	response.OK(c, rule)
}

// UpdateRule 更新排班规则（启用/禁用）
// PUT /api/v1/schedule-rules/:id
func (h *ScheduleRuleHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "规则ID不能为空")
		return
	}

	var req dto.UpdateScheduleRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	rule, err := h.ruleSvc.Update(c.Request.Context(), id, &req, callerID.(string))
	if err != nil {
		h.handleRuleError(c, err)
		return
	}

	response.OK(c, rule)
}

// handleRuleError 统一处理排班规则模块业务错误
func (h *ScheduleRuleHandler) handleRuleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrScheduleRuleNotFound):
		response.NotFound(c, 18001, "排班规则不存在")
	case errors.Is(err, service.ErrScheduleRuleNotConfigurable):
		response.BadRequest(c, 18002, "该规则不可配置")
	default:
		response.InternalError(c)
	}
}
