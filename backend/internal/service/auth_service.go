package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"echo-union/backend/config"
	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
	"echo-union/backend/pkg/jwt"
	"echo-union/backend/pkg/redis"
)

// ── 业务错误 ──

var (
	ErrInvalidCredentials = errors.New("学号或密码错误")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrTokenExpired       = errors.New("Token 已过期")
	ErrTokenInvalid       = errors.New("Token 无效")
	ErrTokenBlacklisted   = errors.New("Token 已被吊销")
	ErrInviteCodeInvalid  = errors.New("邀请码无效或已过期")
	ErrEmailExists        = errors.New("邮箱已被注册")
	ErrStudentIDExists    = errors.New("学号已被注册")
	ErrWeakPassword       = errors.New("密码必须包含字母和数字，长度8-20字符")
	ErrOldPasswordWrong   = errors.New("原密码错误")
)

// passwordRegex 密码强度校验：至少1个字母 + 至少1个数字，8-20字符
// Go regexp 不支持 lookahead，拆分为独立检查
var (
	hasLetter = regexp.MustCompile(`[a-zA-Z]`)
	hasDigit  = regexp.MustCompile(`\d`)
)

// validatePassword 校验密码强度
func validatePassword(password string) bool {
	if len(password) < 8 || len(password) > 20 {
		return false
	}
	return hasLetter.MatchString(password) && hasDigit.MatchString(password)
}

// ── 接口定义 ──

// AuthService 认证业务接口
type AuthService interface {
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.TokenResponse, error)
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error)
	Logout(ctx context.Context, accessJTI string, accessExp time.Time, refreshToken string) error
	GenerateInvite(ctx context.Context, userID string, expiresDays int) (*dto.InviteResponse, error)
	ValidateInvite(ctx context.Context, code string) (*dto.InviteValidateResponse, error)
	ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error
	GetCurrentUser(ctx context.Context, userID string) (*dto.UserDetailResponse, error)
}

// ── 实现 ──

type authService struct {
	cfg    *config.Config
	repo   *repository.Repository
	jwtMgr *jwt.Manager
	rdb    *redis.Client // 可为 nil（降级模式）
	logger *zap.Logger
}

// NewAuthService 创建 AuthService 实例
func NewAuthService(
	cfg *config.Config,
	repo *repository.Repository,
	jwtMgr *jwt.Manager,
	rdb *redis.Client,
	logger *zap.Logger,
) AuthService {
	return &authService{
		cfg:    cfg,
		repo:   repo,
		jwtMgr: jwtMgr,
		rdb:    rdb,
		logger: logger,
	}
}

// ────────────────────── Login ──────────────────────

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
	return s.generateTokenPair(user, req.RememberMe)
}

// ────────────────────── Register ──────────────────────

func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 1. 密码强度校验
	if !validatePassword(req.Password) {
		return nil, ErrWeakPassword
	}

	// 2. 验证邀请码
	invite, err := s.repo.InviteCode.GetByCode(ctx, req.InviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteCodeInvalid
		}
		s.logger.Error("查询邀请码失败", zap.Error(err))
		return nil, err
	}
	if invite.UsedAt != nil || invite.ExpiresAt.Before(time.Now()) {
		return nil, ErrInviteCodeInvalid
	}

	// 3. 检查学号唯一性
	if _, err := s.repo.User.GetByStudentID(ctx, req.StudentID); err == nil {
		return nil, ErrStudentIDExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("检查学号唯一性失败", zap.Error(err))
		return nil, err
	}

	// 4. 检查邮箱唯一性
	if _, err := s.repo.User.GetByEmail(ctx, req.Email); err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("检查邮箱唯一性失败", zap.Error(err))
		return nil, err
	}

	// 5. 验证部门存在
	if _, err := s.repo.Department.GetByID(ctx, req.DepartmentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("部门不存在")
		}
		return nil, err
	}

	// 6. 哈希密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("密码哈希失败", zap.Error(err))
		return nil, err
	}

	// 7. 创建用户
	user := &model.User{
		Name:         req.Name,
		StudentID:    req.StudentID,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "member", // 新注册用户默认为普通成员
		DepartmentID: req.DepartmentID,
	}

	if err := s.repo.User.Create(ctx, user); err != nil {
		s.logger.Error("创建用户失败", zap.Error(err))
		return nil, err
	}

	// 8. 标记邀请码已使用
	if err := s.repo.InviteCode.MarkUsed(ctx, invite.InviteCodeID, user.UserID); err != nil {
		s.logger.Error("标记邀请码已使用失败", zap.Error(err))
		// 用户已创建成功，邀请码标记失败不回滚用户（幂等安全）
	}

	return &dto.RegisterResponse{
		ID:    user.UserID,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

// ────────────────────── RefreshToken ──────────────────────

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	// 1. 解析 Refresh Token
	claims, err := s.jwtMgr.ParseToken(refreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if claims.TokenType != "refresh" {
		return nil, ErrTokenInvalid
	}

	// 2. 检查黑名单
	if s.rdb != nil {
		blacklisted, err := s.rdb.IsBlacklisted(ctx, claims.ID)
		if err != nil {
			s.logger.Warn("Redis 黑名单检查失败，降级放行", zap.Error(err))
		} else if blacklisted {
			return nil, ErrTokenBlacklisted
		}
	}

	// 3. 将旧 Refresh Token 加入黑名单（Token Rotation）
	if s.rdb != nil {
		ttl := time.Until(claims.ExpiresAt.Time)
		if err := s.rdb.BlacklistToken(ctx, claims.ID, ttl); err != nil {
			s.logger.Warn("加入黑名单失败", zap.Error(err))
		}
	}

	// 4. 查询最新用户信息（角色/部门可能已变更）
	user, err := s.repo.User.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("刷新 Token 时查询用户失败", zap.Error(err))
		return nil, err
	}

	// 5. 生成新 Token 对（保持原 RememberMe 设置）
	return s.generateTokenPair(user, claims.RememberMe)
}

// ────────────────────── Logout ──────────────────────

func (s *authService) Logout(ctx context.Context, accessJTI string, accessExp time.Time, refreshToken string) error {
	if s.rdb == nil {
		return nil // 无 Redis 时降级：不做黑名单处理
	}

	// 1. 黑名单 Access Token
	if accessJTI != "" {
		ttl := time.Until(accessExp)
		if err := s.rdb.BlacklistToken(ctx, accessJTI, ttl); err != nil {
			s.logger.Warn("Access Token 加入黑名单失败", zap.Error(err))
		}
	}

	// 2. 黑名单 Refresh Token（如果客户端提供了）
	if refreshToken != "" {
		claims, err := s.jwtMgr.ParseToken(refreshToken)
		if err == nil && claims.TokenType == "refresh" {
			ttl := time.Until(claims.ExpiresAt.Time)
			if err := s.rdb.BlacklistToken(ctx, claims.ID, ttl); err != nil {
				s.logger.Warn("Refresh Token 加入黑名单失败", zap.Error(err))
			}
		}
	}

	return nil
}

// ────────────────────── GenerateInvite ──────────────────────

func (s *authService) GenerateInvite(ctx context.Context, userID string, expiresDays int) (*dto.InviteResponse, error) {
	if expiresDays <= 0 {
		expiresDays = 7
	}

	code, err := generateInviteCode(9)
	if err != nil {
		s.logger.Error("生成邀请码失败", zap.Error(err))
		return nil, err
	}

	expiresAt := time.Now().Add(time.Duration(expiresDays) * 24 * time.Hour)
	createdBy := userID

	invite := &model.InviteCode{
		Code:      code,
		ExpiresAt: expiresAt,
	}
	invite.CreatedBy = &createdBy

	if err := s.repo.InviteCode.Create(ctx, invite); err != nil {
		s.logger.Error("保存邀请码失败", zap.Error(err))
		return nil, err
	}

	inviteURL := fmt.Sprintf("%s/register?code=%s", s.cfg.Server.BaseURL, code)

	return &dto.InviteResponse{
		InviteCode: code,
		InviteURL:  inviteURL,
		ExpiresAt:  expiresAt.Format(time.RFC3339),
	}, nil
}

// ────────────────────── ValidateInvite ──────────────────────

func (s *authService) ValidateInvite(ctx context.Context, code string) (*dto.InviteValidateResponse, error) {
	invite, err := s.repo.InviteCode.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteCodeInvalid
		}
		s.logger.Error("查询邀请码失败", zap.Error(err))
		return nil, err
	}

	if invite.UsedAt != nil || invite.ExpiresAt.Before(time.Now()) {
		return nil, ErrInviteCodeInvalid
	}

	return &dto.InviteValidateResponse{
		Valid:     true,
		ExpiresAt: invite.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// ────────────────────── ChangePassword ──────────────────────

func (s *authService) ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error {
	// 1. 密码强度校验
	if !validatePassword(req.NewPassword) {
		return ErrWeakPassword
	}

	// 2. 查询用户
	user, err := s.repo.User.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// 3. 验证原密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return ErrOldPasswordWrong
	}

	// 4. 哈希新密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("密码哈希失败", zap.Error(err))
		return err
	}

	// 5. 更新
	user.PasswordHash = string(hash)
	user.MustChangePassword = false
	updatedBy := userID
	user.UpdatedBy = &updatedBy

	return s.repo.User.Update(ctx, user)
}

// ────────────────────── GetCurrentUser ──────────────────────

func (s *authService) GetCurrentUser(ctx context.Context, userID string) (*dto.UserDetailResponse, error) {
	user, err := s.repo.User.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("查询当前用户失败", zap.Error(err))
		return nil, err
	}

	var dept *dto.DepartmentResponse
	if user.Department != nil {
		dept = &dto.DepartmentResponse{
			ID:   user.Department.DepartmentID,
			Name: user.Department.Name,
		}
	}

	return &dto.UserDetailResponse{
		ID:         user.UserID,
		Name:       user.Name,
		Email:      user.Email,
		StudentID:  user.StudentID,
		Role:       user.Role,
		Department: dept,
		CreatedAt:  user.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ── 内部辅助方法 ──

// generateTokenPair 为用户生成 Access + Refresh Token 对并构造响应
func (s *authService) generateTokenPair(user *model.User, rememberMe bool) (*dto.TokenResponse, error) {
	accessToken, err := s.jwtMgr.GenerateAccessToken(user.UserID, user.Role, user.DepartmentID)
	if err != nil {
		s.logger.Error("生成 AccessToken 失败", zap.Error(err))
		return nil, err
	}

	refreshToken, err := s.jwtMgr.GenerateRefreshToken(user.UserID, user.Role, user.DepartmentID, rememberMe)
	if err != nil {
		s.logger.Error("生成 RefreshToken 失败", zap.Error(err))
		return nil, err
	}

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

// generateInviteCode 生成加密安全的随机邀请码
func generateInviteCode(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
