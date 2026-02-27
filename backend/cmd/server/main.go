package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"echo-union/backend/config"
	"echo-union/backend/internal/api/handler"
	"echo-union/backend/internal/api/router"
	"echo-union/backend/internal/repository"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/database"
	"echo-union/backend/pkg/jwt"
	applogger "echo-union/backend/pkg/logger"
	"echo-union/backend/pkg/redis"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	logger, err := applogger.NewLogger(&cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("应用启动中...",
		zap.Int("port", cfg.Server.Port),
		zap.String("log_level", cfg.Log.Level),
	)

	// 3. 连接数据库
	db, err := database.NewDB(&cfg.Database, cfg.Log.Level, logger)
	if err != nil {
		logger.Fatal("数据库连接失败", zap.Error(err))
	}
	logger.Info("数据库连接成功")

	// 3.1 执行数据库迁移
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("获取底层 sql.DB 失败", zap.Error(err))
	}
	if err := database.RunMigrations(sqlDB, logger); err != nil {
		logger.Fatal("数据库迁移失败", zap.Error(err))
	}

	// 4. 连接 Redis（可选：连接失败时降级运行，不中断启动）
	var rdb *redis.Client
	rdb, err = redis.NewClient(&cfg.Redis, logger)
	if err != nil {
		logger.Warn("Redis 连接失败，Token 黑名单功能将不可用", zap.Error(err))
		rdb = nil
	}

	// 5. 初始化 JWT 管理器
	jwtMgr := jwt.NewManager(&cfg.Auth)

	// 6. 依赖注入: Repository → Service → Handler
	repo := repository.NewRepository(db)
	svc := service.NewService(cfg, repo, jwtMgr, rdb, logger)
	h := handler.NewHandler(cfg, svc)

	// 7. 初始化路由
	engine := router.Setup(cfg, h, jwtMgr, rdb, db, logger)

	// 8. 启动 HTTP 服务器（优雅关闭）
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("HTTP 服务器已启动", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务器异常", zap.Error(err))
		}
	}()

	// 9. 监听系统信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("收到关闭信号，开始优雅关闭...", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭异常", zap.Error(err))
	}

	// 关闭数据库连接
	closeDB, _ := db.DB()
	if closeDB != nil {
		closeDB.Close()
	}

	// 关闭 Redis 连接
	if rdb != nil {
		rdb.Close()
	}

	logger.Info("服务器已关闭")
}
