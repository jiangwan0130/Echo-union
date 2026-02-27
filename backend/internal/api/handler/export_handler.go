package handler

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// ExportHandler 导出模块 HTTP 处理器
type ExportHandler struct {
	exportSvc service.ExportService
}

// NewExportHandler 创建 ExportHandler
func NewExportHandler(exportSvc service.ExportService) *ExportHandler {
	return &ExportHandler{exportSvc: exportSvc}
}

// ExportSchedule 导出排班表
// GET /api/v1/export/schedule?semester_id=xxx
func (h *ExportHandler) ExportSchedule(c *gin.Context) {
	semesterID := c.Query("semester_id")
	if semesterID == "" {
		response.BadRequest(c, 10001, "semester_id 不能为空")
		return
	}

	buf, filename, err := h.exportSvc.ExportSchedule(c.Request.Context(), semesterID)
	if err != nil {
		h.handleExportError(c, err)
		return
	}

	// 设置下载响应头
	encodedFilename := url.QueryEscape(filename)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+encodedFilename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func (h *ExportHandler) handleExportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrExportNoSchedule):
		response.NotFound(c, 16101, "该学期暂无排班表")
	case errors.Is(err, service.ErrExportNoItems):
		response.BadRequest(c, 16102, "排班表中无排班项")
	case errors.Is(err, service.ErrExportGenerateFail):
		response.InternalError(c)
	default:
		response.InternalError(c)
	}
}
