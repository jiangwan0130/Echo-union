package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 测试辅助 ──

func setupTestScheduleRuleService() (ScheduleRuleService, *mockScheduleRuleRepo) {
	ruleRepo := newMockScheduleRuleRepo()
	repo := &repository.Repository{
		User:         newMockUserRepo(),
		Department:   newMockDeptRepo(),
		Semester:     newMockSemesterRepo(),
		TimeSlot:     newMockTimeSlotRepo(),
		Location:     newMockLocationRepo(),
		SystemConfig: newMockSystemConfigRepo(),
		ScheduleRule: ruleRepo,
	}
	logger := zap.NewNop()
	svc := NewScheduleRuleService(repo, logger)
	return svc, ruleRepo
}

func seedScheduleRules(repo *mockScheduleRuleRepo) {
	repo.rules["rule-R1"] = &model.ScheduleRule{
		RuleID: "rule-R1", RuleCode: "R1", RuleName: "课程冲突检测",
		Description: "不安排与课程冲突的值班", IsEnabled: true, IsConfigurable: true,
	}
	repo.rules["rule-R2"] = &model.ScheduleRule{
		RuleID: "rule-R2", RuleCode: "R2", RuleName: "不可用时间检测",
		Description: "不安排到标记不可用的时段", IsEnabled: true, IsConfigurable: true,
	}
	repo.rules["rule-R6"] = &model.ScheduleRule{
		RuleID: "rule-R6", RuleCode: "R6", RuleName: "核心规则",
		Description: "不可关闭的核心规则", IsEnabled: true, IsConfigurable: false,
	}
}

// ── GetByID 测试 ──

func TestScheduleRuleService_GetByID_Success(t *testing.T) {
	svc, ruleRepo := setupTestScheduleRuleService()
	seedScheduleRules(ruleRepo)

	result, err := svc.GetByID(context.Background(), "rule-R1")
	if err != nil {
		t.Fatalf("GetByID 应成功: %v", err)
	}
	if result.RuleCode != "R1" {
		t.Errorf("期望RuleCode=R1，实际=%s", result.RuleCode)
	}
	if result.RuleName != "课程冲突检测" {
		t.Errorf("期望RuleName=课程冲突检测，实际=%s", result.RuleName)
	}
}

func TestScheduleRuleService_GetByID_NotFound(t *testing.T) {
	svc, _ := setupTestScheduleRuleService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrScheduleRuleNotFound) {
		t.Errorf("期望 ErrScheduleRuleNotFound，实际: %v", err)
	}
}

// ── List 测试 ──

func TestScheduleRuleService_List_Success(t *testing.T) {
	svc, ruleRepo := setupTestScheduleRuleService()
	seedScheduleRules(ruleRepo)

	rules, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}

	if len(rules) != 3 {
		t.Errorf("期望3条规则，实际=%d", len(rules))
	}
}

// ── Update 测试 ──

func TestScheduleRuleService_Update_EnableDisable(t *testing.T) {
	svc, ruleRepo := setupTestScheduleRuleService()
	seedScheduleRules(ruleRepo)

	disabled := false
	req := &dto.UpdateScheduleRuleRequest{IsEnabled: &disabled}

	result, err := svc.Update(context.Background(), "rule-R1", req, "admin-001")
	if err != nil {
		t.Fatalf("Update 应成功: %v", err)
	}
	if result.IsEnabled {
		t.Error("期望规则被禁用")
	}
}

func TestScheduleRuleService_Update_NotConfigurable(t *testing.T) {
	svc, ruleRepo := setupTestScheduleRuleService()
	seedScheduleRules(ruleRepo)

	disabled := false
	req := &dto.UpdateScheduleRuleRequest{IsEnabled: &disabled}

	_, err := svc.Update(context.Background(), "rule-R6", req, "admin-001")
	if !errors.Is(err, ErrScheduleRuleNotConfigurable) {
		t.Errorf("期望 ErrScheduleRuleNotConfigurable，实际: %v", err)
	}
}

func TestScheduleRuleService_Update_NotFound(t *testing.T) {
	svc, _ := setupTestScheduleRuleService()

	disabled := false
	req := &dto.UpdateScheduleRuleRequest{IsEnabled: &disabled}

	_, err := svc.Update(context.Background(), "nonexistent", req, "admin-001")
	if !errors.Is(err, ErrScheduleRuleNotFound) {
		t.Errorf("期望 ErrScheduleRuleNotFound，实际: %v", err)
	}
}
