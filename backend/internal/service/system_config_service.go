package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/repository"
)

// ── 系统配置模块业务错误 ──

var (
	ErrSystemConfigNotFound = errors.New("系统配置未初始化")
)

// SystemConfigService 系统配置业务接口
type SystemConfigService interface {
	Get(ctx context.Context) (*dto.SystemConfigResponse, error)
	Update(ctx context.Context, req *dto.UpdateSystemConfigRequest, callerID string) (*dto.SystemConfigResponse, error)
}

type systemConfigService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewSystemConfigService 创建 SystemConfigService 实例
func NewSystemConfigService(repo *repository.Repository, logger *zap.Logger) SystemConfigService {
	return &systemConfigService{repo: repo, logger: logger}
}

// ────────────────────── Get ──────────────────────

func (s *systemConfigService) Get(ctx context.Context) (*dto.SystemConfigResponse, error) {
	cfg, err := s.repo.SystemConfig.Get(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSystemConfigNotFound
		}
		s.logger.Error("查询系统配置失败", zap.Error(err))
		return nil, err
	}

	return &dto.SystemConfigResponse{
		SwapDeadlineHours:    cfg.SwapDeadlineHours,
		DutyReminderTime:     cfg.DutyReminderTime,
		DefaultLocation:      cfg.DefaultLocation,
		SignInWindowMinutes:  cfg.SignInWindowMinutes,
		SignOutWindowMinutes: cfg.SignOutWindowMinutes,
		UpdatedAt:            cfg.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// ────────────────────── Update ──────────────────────

func (s *systemConfigService) Update(ctx context.Context, req *dto.UpdateSystemConfigRequest, callerID string) (*dto.SystemConfigResponse, error) {
	cfg, err := s.repo.SystemConfig.Get(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSystemConfigNotFound
		}
		s.logger.Error("查询系统配置失败", zap.Error(err))
		return nil, err
	}

	if req.SwapDeadlineHours != nil {
		cfg.SwapDeadlineHours = *req.SwapDeadlineHours
	}
	if req.DutyReminderTime != nil {
		cfg.DutyReminderTime = *req.DutyReminderTime
	}
	if req.DefaultLocation != nil {
		cfg.DefaultLocation = *req.DefaultLocation
	}
	if req.SignInWindowMinutes != nil {
		cfg.SignInWindowMinutes = *req.SignInWindowMinutes
	}
	if req.SignOutWindowMinutes != nil {
		cfg.SignOutWindowMinutes = *req.SignOutWindowMinutes
	}

	cfg.UpdatedBy = &callerID

	if err := s.repo.SystemConfig.Update(ctx, cfg); err != nil {
		s.logger.Error("更新系统配置失败", zap.Error(err))
		return nil, err
	}

	return &dto.SystemConfigResponse{
		SwapDeadlineHours:    cfg.SwapDeadlineHours,
		DutyReminderTime:     cfg.DutyReminderTime,
		DefaultLocation:      cfg.DefaultLocation,
		SignInWindowMinutes:  cfg.SignInWindowMinutes,
		SignOutWindowMinutes: cfg.SignOutWindowMinutes,
		UpdatedAt:            cfg.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
