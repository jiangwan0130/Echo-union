package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"echo-union/backend/config"
	"echo-union/backend/internal/api/handler"
	"echo-union/backend/internal/api/middleware"
	"echo-union/backend/pkg/jwt"
)

// Setup 初始化并返回 Gin 路由引擎
func Setup(cfg *config.Config, h *handler.Handler, jwtMgr *jwt.Manager, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// ── 全局中间件 ──
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.Server.CORS.AllowOrigins))

	// ── 健康检查 ──
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ── API v1 ──
	v1 := r.Group("/api/v1")
	{
		// 认证模块（无需认证）
		auth := v1.Group("/auth")
		{
			auth.POST("/login", h.Auth.Login)
			auth.POST("/register", h.Auth.Register)
			auth.POST("/refresh", h.Auth.RefreshToken)
			auth.GET("/invite/:code", h.Auth.ValidateInvite)
		}

		// 需要认证的路由
		authorized := v1.Group("")
		authorized.Use(middleware.JWTAuth(jwtMgr))
		{
			// 认证模块（需要认证）
			authorized.POST("/auth/logout", h.Auth.Logout)
			authorized.POST("/auth/invite", middleware.RoleAuth("admin", "leader"), h.Auth.GenerateInvite)

			// 用户模块
			users := authorized.Group("/users")
			{
				users.GET("/me", h.User.GetCurrentUser)
				users.GET("", middleware.RoleAuth("admin", "leader"), h.User.ListUsers)
			}

			// 📝 后续按模块扩展路由组:
			// /departments, /semesters, /schedules, /swaps, /duties, /notifications 等
		}
	}

	return r
}

// [自证通过] internal/api/router/router.go
