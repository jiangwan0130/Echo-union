package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// TimetableHandler 时间表模块 Handler
type TimetableHandler struct {
	svc service.TimetableService
}

// NewTimetableHandler 创建 TimetableHandler 实例
func NewTimetableHandler(svc service.TimetableService) *TimetableHandler {
	return &TimetableHandler{svc: svc}
}

// ImportICS 导入 ICS 课表
// POST /api/v1/timetables/import
//
// 支持两种方式：
//   - 文件上传: multipart/form-data, field="file"
//   - URL 导入: application/json, body={"url": "..."}
func (h *TimetableHandler) ImportICS(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 获取可选的 semester_id
	semesterID := c.PostForm("semester_id")

	// 尝试文件上传方式
	file, _, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		resp, err := h.svc.ImportICS(c.Request.Context(), file, userID.(string), semesterID)
		if err != nil {
			handleTimetableError(c, err)
			return
		}
		response.Created(c, resp)
		return
	}

	// 尝试 URL 方式
	var req dto.ImportICSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 也可能是纯 form 提交
		req.URL = c.PostForm("url")
		if req.URL == "" {
			response.BadRequest(c, 15000, "请上传 ICS 文件或提供 ICS URL")
			return
		}
		if semesterID == "" {
			semesterID = c.PostForm("semester_id")
		}
	}
	if req.SemesterID != "" {
		semesterID = req.SemesterID
	}

	if req.URL == "" {
		response.BadRequest(c, 15000, "请上传 ICS 文件或提供 ICS URL")
		return
	}

	// 获取 ICS 内容
	body, err := service.FetchICSContent(req.URL)
	if err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, 15001, "ICS URL 获取失败", err.Error())
		return
	}
	defer body.Close()

	resp, err := h.svc.ImportICS(c.Request.Context(), body, userID.(string), semesterID)
	if err != nil {
		handleTimetableError(c, err)
		return
	}
	response.Created(c, resp)
}

// GetMyTimetable 获取我的时间表
// GET /api/v1/timetables/me
func (h *TimetableHandler) GetMyTimetable(c *gin.Context) {
	userID, _ := c.Get("user_id")
	semesterID := c.Query("semester_id")

	resp, err := h.svc.GetMyTimetable(c.Request.Context(), userID.(string), semesterID)
	if err != nil {
		handleTimetableError(c, err)
		return
	}
	response.OK(c, resp)
}

// CreateUnavailableTime 添加不可用时间
// POST /api/v1/timetables/unavailable
func (h *TimetableHandler) CreateUnavailableTime(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req dto.CreateUnavailableTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 15000, err.Error())
		return
	}

	resp, err := h.svc.CreateUnavailableTime(c.Request.Context(), &req, userID.(string))
	if err != nil {
		handleTimetableError(c, err)
		return
	}
	response.Created(c, resp)
}

// UpdateUnavailableTime 更新不可用时间
// PUT /api/v1/timetables/unavailable/:id
func (h *TimetableHandler) UpdateUnavailableTime(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	var req dto.UpdateUnavailableTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 15000, err.Error())
		return
	}

	resp, err := h.svc.UpdateUnavailableTime(c.Request.Context(), id, &req, userID.(string))
	if err != nil {
		handleTimetableError(c, err)
		return
	}
	response.OK(c, resp)
}

// DeleteUnavailableTime 删除不可用时间
// DELETE /api/v1/timetables/unavailable/:id
func (h *TimetableHandler) DeleteUnavailableTime(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	err := h.svc.DeleteUnavailableTime(c.Request.Context(), id, userID.(string))
	if err != nil {
		handleTimetableError(c, err)
		return
	}
	response.OK(c, nil)
}

// SubmitTimetable 提交时间表
// POST /api/v1/timetables/submit
func (h *TimetableHandler) SubmitTimetable(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req dto.SubmitTimetableRequest
	// body 可为空，semester_id 可选
	_ = c.ShouldBindJSON(&req)

	resp, err := h.svc.SubmitTimetable(c.Request.Context(), userID.(string), req.SemesterID)
	if err != nil {
		handleTimetableError(c, err)
		return
	}
	response.OK(c, resp)
}

// GetProgress 获取全局提交进度
// GET /api/v1/timetables/progress
func (h *TimetableHandler) GetProgress(c *gin.Context) {
	semesterID := c.Query("semester_id")

	resp, err := h.svc.GetProgress(c.Request.Context(), semesterID)
	if err != nil {
		handleTimetableError(c, err)
		return
	}
	response.OK(c, resp)
}

// GetDepartmentProgress 获取部门提交进度
// GET /api/v1/timetables/progress/department/:id
func (h *TimetableHandler) GetDepartmentProgress(c *gin.Context) {
	departmentID := c.Param("id")
	semesterID := c.Query("semester_id")

	resp, err := h.svc.GetDepartmentProgress(c.Request.Context(), departmentID, semesterID)
	if err != nil {
		handleTimetableError(c, err)
		return
	}
	response.OK(c, resp)
}

// handleTimetableError 统一时间表模块错误映射
func handleTimetableError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrTimetableNoActiveSemester):
		response.ErrorWithDetails(c, http.StatusBadRequest, 15002, "无活动学期", err.Error())
	case errors.Is(err, service.ErrTimetableAssignmentNotFound):
		response.ErrorWithDetails(c, http.StatusNotFound, 15003, "未找到学期分配记录", err.Error())
	case errors.Is(err, service.ErrTimetableAlreadySubmitted):
		response.ErrorWithDetails(c, http.StatusConflict, 15004, "时间表已提交", err.Error())
	case errors.Is(err, service.ErrTimetableNoCourses):
		response.ErrorWithDetails(c, http.StatusBadRequest, 15005, "尚未导入课表或标记不可用时间", err.Error())
	case errors.Is(err, service.ErrTimetableICSParseFailed):
		response.ErrorWithDetails(c, http.StatusBadRequest, 15006, "ICS 文件解析失败", err.Error())
	case errors.Is(err, service.ErrTimetableICSEmpty):
		response.ErrorWithDetails(c, http.StatusBadRequest, 15007, "ICS 文件中无有效课程", err.Error())
	case errors.Is(err, service.ErrTimetableUnavailableNotFound):
		response.NotFound(c, 15008, err.Error())
	case errors.Is(err, service.ErrTimetableUnavailableNotOwner):
		response.ErrorWithDetails(c, http.StatusForbidden, 15009, "无权操作", err.Error())
	case errors.Is(err, service.ErrTimetableDepartmentNotFound):
		response.NotFound(c, 15010, err.Error())
	default:
		response.InternalError(c)
	}
}
