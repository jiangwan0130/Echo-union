package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// LocationHandler 地点模块 HTTP 处理器
type LocationHandler struct {
	locationSvc service.LocationService
}

// NewLocationHandler 创建 LocationHandler
func NewLocationHandler(locationSvc service.LocationService) *LocationHandler {
	return &LocationHandler{locationSvc: locationSvc}
}

// ListLocations 获取地点列表
// GET /api/v1/locations
func (h *LocationHandler) ListLocations(c *gin.Context) {
	var req dto.LocationListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	locations, err := h.locationSvc.List(c.Request.Context(), &req)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"list": locations})
}

// GetLocation 获取地点详情
// GET /api/v1/locations/:id
func (h *LocationHandler) GetLocation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "地点ID不能为空")
		return
	}

	location, err := h.locationSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleLocationError(c, err)
		return
	}

	response.OK(c, location)
}

// CreateLocation 创建地点
// POST /api/v1/locations
func (h *LocationHandler) CreateLocation(c *gin.Context) {
	var req dto.CreateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	location, err := h.locationSvc.Create(c.Request.Context(), &req, callerID)
	if err != nil {
		h.handleLocationError(c, err)
		return
	}

	response.Created(c, location)
}

// UpdateLocation 更新地点
// PUT /api/v1/locations/:id
func (h *LocationHandler) UpdateLocation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "地点ID不能为空")
		return
	}

	var req dto.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	location, err := h.locationSvc.Update(c.Request.Context(), id, &req, callerID)
	if err != nil {
		h.handleLocationError(c, err)
		return
	}

	response.OK(c, location)
}

// DeleteLocation 删除地点
// DELETE /api/v1/locations/:id
func (h *LocationHandler) DeleteLocation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, 10001, "地点ID不能为空")
		return
	}

	callerID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	if err := h.locationSvc.Delete(c.Request.Context(), id, callerID); err != nil {
		h.handleLocationError(c, err)
		return
	}

	response.OK(c, nil)
}

// handleLocationError 统一处理地点模块业务错误
func (h *LocationHandler) handleLocationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLocationNotFound):
		response.NotFound(c, 16001, "地点不存在")
	default:
		response.InternalError(c)
	}
}
