package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 排班规则模块业务错误 ──

var (
	ErrScheduleRuleNotFound        = errors.New("排班规则不存在")
	ErrScheduleRuleNotConfigurable = errors.New("该规则不可配置")
)

// ScheduleRuleService 排班规则业务接口
type ScheduleRuleService interface {
	GetByID(ctx context.Context, id string) (*dto.ScheduleRuleResponse, error)
	List(ctx context.Context) ([]dto.ScheduleRuleResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateScheduleRuleRequest, callerID string) (*dto.ScheduleRuleResponse, error)
}

type scheduleRuleService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewScheduleRuleService 创建 ScheduleRuleService 实例
func NewScheduleRuleService(repo *repository.Repository, logger *zap.Logger) ScheduleRuleService {
	return &scheduleRuleService{repo: repo, logger: logger}
}

// ────────────────────── GetByID ──────────────────────

func (s *scheduleRuleService) GetByID(ctx context.Context, id string) (*dto.ScheduleRuleResponse, error) {
	rule, err := s.repo.ScheduleRule.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleRuleNotFound
		}
		s.logger.Error("查询排班规则失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toScheduleRuleResponse(rule), nil
}

// ────────────────────── List ──────────────────────

func (s *scheduleRuleService) List(ctx context.Context) ([]dto.ScheduleRuleResponse, error) {
	rules, err := s.repo.ScheduleRule.List(ctx)
	if err != nil {
		s.logger.Error("列出排班规则失败", zap.Error(err))
		return nil, err
	}

	result := make([]dto.ScheduleRuleResponse, 0, len(rules))
	for i := range rules {
		result = append(result, *s.toScheduleRuleResponse(&rules[i]))
	}

	return result, nil
}

// ────────────────────── Update ──────────────────────

func (s *scheduleRuleService) Update(ctx context.Context, id string, req *dto.UpdateScheduleRuleRequest, callerID string) (*dto.ScheduleRuleResponse, error) {
	rule, err := s.repo.ScheduleRule.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleRuleNotFound
		}
		s.logger.Error("查询排班规则失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	// 检查规则是否可配置
	if !rule.IsConfigurable {
		return nil, ErrScheduleRuleNotConfigurable
	}

	if req.IsEnabled != nil {
		rule.IsEnabled = *req.IsEnabled
	}

	rule.UpdatedBy = &callerID

	if err := s.repo.ScheduleRule.Update(ctx, rule); err != nil {
		s.logger.Error("更新排班规则失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toScheduleRuleResponse(rule), nil
}

// ── 内部辅助方法 ──

func (s *scheduleRuleService) toScheduleRuleResponse(rule *model.ScheduleRule) *dto.ScheduleRuleResponse {
	return &dto.ScheduleRuleResponse{
		ID:             rule.RuleID,
		RuleCode:       rule.RuleCode,
		RuleName:       rule.RuleName,
		Description:    rule.Description,
		IsEnabled:      rule.IsEnabled,
		IsConfigurable: rule.IsConfigurable,
		CreatedAt:      rule.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      rule.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
