package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// SystemConfigHandler 系统配置模块 HTTP 处理器
type SystemConfigHandler struct {
	configSvc service.SystemConfigService
}

// NewSystemConfigHandler 创建 SystemConfigHandler
func NewSystemConfigHandler(configSvc service.SystemConfigService) *SystemConfigHandler {
	return &SystemConfigHandler{configSvc: configSvc}
}

// GetConfig 获取系统配置
// GET /api/v1/system-config
func (h *SystemConfigHandler) GetConfig(c *gin.Context) {
	cfg, err := h.configSvc.Get(c.Request.Context())
	if err != nil {
		h.handleConfigError(c, err)
		return
	}

	response.OK(c, cfg)
}

// UpdateConfig 更新系统配置
// PUT /api/v1/system-config
func (h *SystemConfigHandler) UpdateConfig(c *gin.Context) {
	var req dto.UpdateSystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, _ := c.Get("user_id")

	cfg, err := h.configSvc.Update(c.Request.Context(), &req, callerID.(string))
	if err != nil {
		h.handleConfigError(c, err)
		return
	}

	response.OK(c, cfg)
}

// handleConfigError 统一处理系统配置模块业务错误
func (h *SystemConfigHandler) handleConfigError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSystemConfigNotFound):
		response.NotFound(c, 17001, "系统配置未初始化")
	default:
		response.InternalError(c)
	}
}
