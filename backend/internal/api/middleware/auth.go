package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"echo-union/backend/pkg/jwt"
	"echo-union/backend/pkg/redis"
	"echo-union/backend/pkg/response"
)

// JWTAuth JWT 认证中间件
// 从 Authorization: Bearer <token> 中提取并验证 Access Token
// rdb 可为 nil（Redis 不可用时降级：跳过黑名单检查）
func JWTAuth(jwtMgr *jwt.Manager, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, 10002, "缺少认证头")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, 10002, "认证头格式无效")
			c.Abort()
			return
		}

		claims, err := jwtMgr.ParseToken(parts[1])
		if err != nil {
			response.Unauthorized(c, 10002, "Token 无效或已过期")
			c.Abort()
			return
		}

		if claims.TokenType != "access" {
			response.Unauthorized(c, 10002, "Token 类型无效")
			c.Abort()
			return
		}

		// 检查 Token 黑名单（Redis 可用时）
		if rdb != nil {
			blacklisted, err := rdb.IsBlacklisted(c.Request.Context(), claims.ID)
			if err == nil && blacklisted {
				response.Unauthorized(c, 11003, "Token 已被吊销")
				c.Abort()
				return
			}
			// Redis 出错时降级放行（fail-open），日志由 Redis 客户端处理
		}

		// 将用户信息注入上下文
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("department_id", claims.DepartmentID)
		// Logout 需要 JTI 和过期时间用于加入黑名单
		c.Set("token_jti", claims.ID)
		if claims.ExpiresAt != nil {
			c.Set("token_exp", claims.ExpiresAt.Time)
		}

		c.Next()
	}
}

// RoleAuth 角色权限中间件
// 检查当前用户是否具有指定角色之一
func RoleAuth(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Unauthorized(c, 10002, "未认证")
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, r := range allowedRoles {
			if userRole == r {
				c.Next()
				return
			}
		}

		response.Forbidden(c, 10003, "无权限访问")
		c.Abort()
	}
}
