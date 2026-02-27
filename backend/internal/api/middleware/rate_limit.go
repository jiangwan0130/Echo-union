package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"echo-union/backend/pkg/redis"
	"echo-union/backend/pkg/response"
)

// RateLimit 基于 Redis 滑动窗口的速率限制中间件
// limit: 窗口内允许的最大请求数
// window: 滑动窗口时长
// rdb 为 nil 时降级放行（与 JWTAuth 策略一致）
func RateLimit(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		key := fmt.Sprintf("rate_limit:%s:%s", c.ClientIP(), c.FullPath())
		allowed, err := rdb.CheckRateLimit(c.Request.Context(), key, limit, window)
		if err != nil {
			// Redis 出错时降级放行
			c.Next()
			return
		}

		if !allowed {
			response.Error(c, http.StatusTooManyRequests, 10004, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
