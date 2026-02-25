package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/repository"
)

// ── 测试辅助 ──

func setupTestSystemConfigService() (SystemConfigService, *mockSystemConfigRepo) {
	configRepo := newMockSystemConfigRepo()
	repo := &repository.Repository{
		User:         newMockUserRepo(),
		Department:   newMockDeptRepo(),
		InviteCode:   newMockInviteCodeRepo(),
		Semester:     newMockSemesterRepo(),
		TimeSlot:     newMockTimeSlotRepo(),
		Location:     newMockLocationRepo(),
		SystemConfig: configRepo,
		ScheduleRule: newMockScheduleRuleRepo(),
	}
	logger := zap.NewNop()
	svc := NewSystemConfigService(repo, logger)
	return svc, configRepo
}

// ── Get 测试 ──

func TestSystemConfigService_Get_Success(t *testing.T) {
	svc, _ := setupTestSystemConfigService()

	result, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get 应成功: %v", err)
	}
	if result.SwapDeadlineHours != 24 {
		t.Errorf("期望SwapDeadlineHours=24，实际=%d", result.SwapDeadlineHours)
	}
	if result.DefaultLocation != "学生会办公室" {
		t.Errorf("期望DefaultLocation=学生会办公室，实际=%s", result.DefaultLocation)
	}
}

func TestSystemConfigService_Get_NotFound(t *testing.T) {
	svc, configRepo := setupTestSystemConfigService()
	configRepo.cfg = nil

	_, err := svc.Get(context.Background())
	if !errors.Is(err, ErrSystemConfigNotFound) {
		t.Errorf("期望 ErrSystemConfigNotFound，实际: %v", err)
	}
}

// ── Update 测试 ──

func TestSystemConfigService_Update_Success(t *testing.T) {
	svc, _ := setupTestSystemConfigService()

	newHours := 48
	newLocation := "新办公室"
	req := &dto.UpdateSystemConfigRequest{
		SwapDeadlineHours: &newHours,
		DefaultLocation:   &newLocation,
	}

	result, err := svc.Update(context.Background(), req, "admin-001")
	if err != nil {
		t.Fatalf("Update 应成功: %v", err)
	}
	if result.SwapDeadlineHours != 48 {
		t.Errorf("期望SwapDeadlineHours=48，实际=%d", result.SwapDeadlineHours)
	}
	if result.DefaultLocation != "新办公室" {
		t.Errorf("期望DefaultLocation=新办公室，实际=%s", result.DefaultLocation)
	}
	// 未修改的字段应保持原值
	if result.SignInWindowMinutes != 15 {
		t.Errorf("期望SignInWindowMinutes=15（未修改），实际=%d", result.SignInWindowMinutes)
	}
}

func TestSystemConfigService_Update_PartialUpdate(t *testing.T) {
	svc, _ := setupTestSystemConfigService()

	newMinutes := 30
	req := &dto.UpdateSystemConfigRequest{
		SignInWindowMinutes: &newMinutes,
	}

	result, err := svc.Update(context.Background(), req, "admin-001")
	if err != nil {
		t.Fatalf("Update 应成功: %v", err)
	}
	if result.SignInWindowMinutes != 30 {
		t.Errorf("期望SignInWindowMinutes=30，实际=%d", result.SignInWindowMinutes)
	}
	// 其他字段应保持不变
	if result.SwapDeadlineHours != 24 {
		t.Errorf("期望SwapDeadlineHours=24（未修改），实际=%d", result.SwapDeadlineHours)
	}
}

func TestSystemConfigService_Update_NotFound(t *testing.T) {
	svc, configRepo := setupTestSystemConfigService()
	configRepo.cfg = nil

	newHours := 48
	req := &dto.UpdateSystemConfigRequest{SwapDeadlineHours: &newHours}

	_, err := svc.Update(context.Background(), req, "admin-001")
	if !errors.Is(err, ErrSystemConfigNotFound) {
		t.Errorf("期望 ErrSystemConfigNotFound，实际: %v", err)
	}
}
