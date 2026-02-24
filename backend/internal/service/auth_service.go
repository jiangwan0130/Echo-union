package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"echo-union/backend/config"
	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/repository"
	"echo-union/backend/pkg/jwt"
)

var (
	ErrInvalidCredentials = errors.New("学号或密码错误")
	ErrUserNotFound       = errors.New("用户不存在")
)

// AuthService 认证业务接口
type AuthService interface {
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.TokenResponse, error)
	// 📝 按需扩展: Logout, RefreshToken, Register, GenerateInvite, ValidateInvite 等
}

type authService struct {
	cfg    *config.Config
	repo   *repository.Repository
	jwtMgr *jwt.Manager
	logger *zap.Logger
}

// NewAuthService 创建 AuthService 实例
func NewAuthService(
	cfg *config.Config,
	repo *repository.Repository,
	jwtMgr *jwt.Manager,
	logger *zap.Logger,
) AuthService {
	return &authService{
		cfg:    cfg,
		repo:   repo,
		jwtMgr: jwtMgr,
		logger: logger,
	}
}

func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.TokenResponse, error) {
	// 1. 查询用户
	user, err := s.repo.User.GetByStudentID(ctx, req.StudentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		s.logger.Error("查询用户失败", zap.Error(err))
		return nil, err
	}

	// 2. 验证密码 (bcrypt)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 3. 生成 Token 对
	accessToken, err := s.jwtMgr.GenerateAccessToken(user.UserID, user.Role, user.DepartmentID)
	if err != nil {
		s.logger.Error("生成 AccessToken 失败", zap.Error(err))
		return nil, err
	}

	refreshToken, err := s.jwtMgr.GenerateRefreshToken(user.UserID, user.Role, user.DepartmentID, req.RememberMe)
	if err != nil {
		s.logger.Error("生成 RefreshToken 失败", zap.Error(err))
		return nil, err
	}

	// 4. 构造响应
	var dept *dto.DepartmentResponse
	if user.Department != nil {
		dept = &dto.DepartmentResponse{
			ID:   user.Department.DepartmentID,
			Name: user.Department.Name,
		}
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.Auth.AccessTokenTTL.Seconds()),
		User: dto.UserResponse{
			ID:         user.UserID,
			Name:       user.Name,
			Email:      user.Email,
			StudentID:  user.StudentID,
			Role:       user.Role,
			Department: dept,
		},
	}, nil
}

// [自证通过] internal/service/auth_service.go
