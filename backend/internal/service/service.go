package service

import (
	"go.uber.org/zap"

	"echo-union/backend/config"
	"echo-union/backend/internal/repository"
	"echo-union/backend/pkg/jwt"
)

// Service 所有 Service 的聚合入口
type Service struct {
	Auth AuthService
	User UserService
	// 📝 后续按模块扩展: Schedule, Swap, Duty, Notification 等
}

// NewService 创建 Service 聚合
func NewService(
	cfg *config.Config,
	repo *repository.Repository,
	jwtMgr *jwt.Manager,
	logger *zap.Logger,
) *Service {
	return &Service{
		Auth: NewAuthService(cfg, repo, jwtMgr, logger),
		User: NewUserService(repo, logger),
	}
}

// [自证通过] internal/service/service.go
