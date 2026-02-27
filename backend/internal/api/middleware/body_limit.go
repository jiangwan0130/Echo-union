package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"echo-union/backend/pkg/response"
)

// BodyLimit 全局请求体大小限制中间件
// maxBytes: 允许的最大请求体字节数（如 1<<20 = 1MB）
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}

		c.Next()

		// 检查是否因为超出限制而失败
		if c.IsAborted() {
			return
		}
		for _, err := range c.Errors {
			if err.Err != nil && err.Err.Error() == "http: request body too large" {
				response.Error(c, http.StatusRequestEntityTooLarge, 10005, "请求体过大")
				return
			}
		}
	}
}
