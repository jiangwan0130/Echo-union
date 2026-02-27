package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "request_id"

// RequestID 请求追踪 ID 中间件
// 从请求头 X-Request-ID 读取，若不存在则自动生成 UUID
// 结果注入到 gin.Context 中并设置响应头 X-Request-ID
// requestIDMaxLen 限制外部传入的 Request-ID 最大长度，防止日志注入
const requestIDMaxLen = 64

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" || len(rid) > requestIDMaxLen {
			rid = uuid.New().String()
		}

		c.Set(requestIDKey, rid)
		c.Header("X-Request-ID", rid)

		c.Next()
	}
}
