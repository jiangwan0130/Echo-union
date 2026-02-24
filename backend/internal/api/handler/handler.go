package handler

import "echo-union/backend/internal/service"

// Handler 所有 Handler 的聚合入口
type Handler struct {
	Auth *AuthHandler
	User *UserHandler
	// 📝 后续按模块扩展: Schedule, Swap, Duty, Notification 等
}

// NewHandler 创建 Handler 聚合
func NewHandler(svc *service.Service) *Handler {
	return &Handler{
		Auth: NewAuthHandler(svc.Auth),
		User: NewUserHandler(svc.User),
	}
}

// [自证通过] internal/api/handler/handler.go
