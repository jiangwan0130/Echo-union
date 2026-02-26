package handler

import (
	"github.com/gin-gonic/gin"

	"echo-union/backend/pkg/response"
)

// MustGetUserID 从 Gin 上下文中安全提取 user_id。
// 如果 JWT 中间件未正确注入 user_id，返回 false 并写入 401 响应。
// 调用方应在 ok=false 时直接 return。
func MustGetUserID(c *gin.Context) (string, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, 10002, "未认证")
		return "", false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		response.Unauthorized(c, 10002, "未认证")
		return "", false
	}
	return s, true
}

// MustGetRole 从 Gin 上下文中安全提取 role。
func MustGetRole(c *gin.Context) (string, bool) {
	v, exists := c.Get("role")
	if !exists {
		response.Unauthorized(c, 10002, "未认证")
		return "", false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		response.Unauthorized(c, 10002, "未认证")
		return "", false
	}
	return s, true
}

// MustGetDepartmentID 从 Gin 上下文中安全提取 department_id。
func MustGetDepartmentID(c *gin.Context) (string, bool) {
	v, exists := c.Get("department_id")
	if !exists {
		response.Unauthorized(c, 10002, "未认证")
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		response.Unauthorized(c, 10002, "未认证")
		return "", false
	}
	return s, true
}
