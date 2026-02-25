package repository

import (
	"context"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// ScheduleRuleRepository 排班规则数据访问接口
type ScheduleRuleRepository interface {
	GetByID(ctx context.Context, id string) (*model.ScheduleRule, error)
	List(ctx context.Context) ([]model.ScheduleRule, error)
	Update(ctx context.Context, rule *model.ScheduleRule) error
}

type scheduleRuleRepo struct {
	db *gorm.DB
}

// NewScheduleRuleRepo 创建 ScheduleRuleRepository 实例
func NewScheduleRuleRepo(db *gorm.DB) ScheduleRuleRepository {
	return &scheduleRuleRepo{db: db}
}

func (r *scheduleRuleRepo) GetByID(ctx context.Context, id string) (*model.ScheduleRule, error) {
	var rule model.ScheduleRule
	err := r.db.WithContext(ctx).
		Where("rule_id = ?", id).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *scheduleRuleRepo) List(ctx context.Context) ([]model.ScheduleRule, error) {
	var rules []model.ScheduleRule
	err := r.db.WithContext(ctx).
		Order("rule_code ASC").
		Find(&rules).Error
	return rules, err
}

func (r *scheduleRuleRepo) Update(ctx context.Context, rule *model.ScheduleRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}
