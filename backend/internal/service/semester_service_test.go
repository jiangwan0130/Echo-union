package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 测试辅助 ──

func setupTestSemesterService() (SemesterService, *mockSemesterRepo) {
	semesterRepo := newMockSemesterRepo()
	repo := &repository.Repository{
		User:         newMockUserRepo(),
		Department:   newMockDeptRepo(),
		InviteCode:   newMockInviteCodeRepo(),
		Semester:     semesterRepo,
		TimeSlot:     newMockTimeSlotRepo(),
		Location:     newMockLocationRepo(),
		SystemConfig: newMockSystemConfigRepo(),
		ScheduleRule: newMockScheduleRuleRepo(),
	}
	logger := zap.NewNop()
	svc := NewSemesterService(repo, logger)
	return svc, semesterRepo
}

// ── Create 测试 ──

func TestSemesterService_Create_Success(t *testing.T) {
	svc, _ := setupTestSemesterService()

	req := &dto.CreateSemesterRequest{
		Name:          "2025-2026学年第二学期",
		StartDate:     "2026-02-20",
		EndDate:       "2026-07-10",
		FirstWeekType: "odd",
	}

	result, err := svc.Create(context.Background(), req, "admin-001")
	if err != nil {
		t.Fatalf("Create 应成功: %v", err)
	}
	if result.Name != "2025-2026学年第二学期" {
		t.Errorf("期望Name=2025-2026学年第二学期，实际=%s", result.Name)
	}
	if result.IsActive {
		t.Error("新创建学期不应默认激活")
	}
	if result.Status != "active" {
		t.Errorf("期望Status=active，实际=%s", result.Status)
	}
}

func TestSemesterService_Create_InvalidDate(t *testing.T) {
	svc, _ := setupTestSemesterService()

	// 结束日期早于开始日期
	req := &dto.CreateSemesterRequest{
		Name:          "测试学期",
		StartDate:     "2026-07-10",
		EndDate:       "2026-02-20",
		FirstWeekType: "odd",
	}

	_, err := svc.Create(context.Background(), req, "admin-001")
	if !errors.Is(err, ErrSemesterDateInvalid) {
		t.Errorf("期望 ErrSemesterDateInvalid，实际: %v", err)
	}
}

func TestSemesterService_Create_BadDateFormat(t *testing.T) {
	svc, _ := setupTestSemesterService()

	req := &dto.CreateSemesterRequest{
		Name:          "测试学期",
		StartDate:     "invalid-date",
		EndDate:       "2026-07-10",
		FirstWeekType: "odd",
	}

	_, err := svc.Create(context.Background(), req, "admin-001")
	if !errors.Is(err, ErrSemesterDateInvalid) {
		t.Errorf("期望 ErrSemesterDateInvalid，实际: %v", err)
	}
}

// ── GetByID 测试 ──

func TestSemesterService_GetByID_Success(t *testing.T) {
	svc, semesterRepo := setupTestSemesterService()
	semesterRepo.semesters["sem-001"] = &model.Semester{
		SemesterID:    "sem-001",
		Name:          "测试学期",
		StartDate:     time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		FirstWeekType: "odd",
		IsActive:      true,
		Status:        "active",
	}

	result, err := svc.GetByID(context.Background(), "sem-001")
	if err != nil {
		t.Fatalf("GetByID 应成功: %v", err)
	}
	if result.Name != "测试学期" {
		t.Errorf("期望Name=测试学期，实际=%s", result.Name)
	}
}

func TestSemesterService_GetByID_NotFound(t *testing.T) {
	svc, _ := setupTestSemesterService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrSemesterNotFound) {
		t.Errorf("期望 ErrSemesterNotFound，实际: %v", err)
	}
}

// ── GetCurrent 测试 ──

func TestSemesterService_GetCurrent_Success(t *testing.T) {
	svc, semesterRepo := setupTestSemesterService()
	semesterRepo.semesters["sem-001"] = &model.Semester{
		SemesterID:    "sem-001",
		Name:          "当前学期",
		StartDate:     time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		FirstWeekType: "odd",
		IsActive:      true,
		Status:        "active",
	}

	result, err := svc.GetCurrent(context.Background())
	if err != nil {
		t.Fatalf("GetCurrent 应成功: %v", err)
	}
	if result.Name != "当前学期" {
		t.Errorf("期望Name=当前学期，实际=%s", result.Name)
	}
}

func TestSemesterService_GetCurrent_NotFound(t *testing.T) {
	svc, _ := setupTestSemesterService()

	_, err := svc.GetCurrent(context.Background())
	if !errors.Is(err, ErrSemesterNotFound) {
		t.Errorf("期望 ErrSemesterNotFound，实际: %v", err)
	}
}

// ── Activate 测试 ──

func TestSemesterService_Activate_Success(t *testing.T) {
	svc, semesterRepo := setupTestSemesterService()
	semesterRepo.semesters["sem-001"] = &model.Semester{
		SemesterID:    "sem-001",
		Name:          "学期A",
		StartDate:     time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		FirstWeekType: "odd",
		IsActive:      true,
		Status:        "active",
	}
	semesterRepo.semesters["sem-002"] = &model.Semester{
		SemesterID:    "sem-002",
		Name:          "学期B",
		StartDate:     time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC),
		FirstWeekType: "even",
		IsActive:      false,
		Status:        "active",
	}

	err := svc.Activate(context.Background(), "sem-002", "admin-001")
	if err != nil {
		t.Fatalf("Activate 应成功: %v", err)
	}

	// sem-001 应被取消激活
	if semesterRepo.semesters["sem-001"].IsActive {
		t.Error("sem-001 应被取消激活")
	}
	// sem-002 应被激活
	if !semesterRepo.semesters["sem-002"].IsActive {
		t.Error("sem-002 应被激活")
	}
}

func TestSemesterService_Activate_NotFound(t *testing.T) {
	svc, _ := setupTestSemesterService()

	err := svc.Activate(context.Background(), "nonexistent", "admin-001")
	if !errors.Is(err, ErrSemesterNotFound) {
		t.Errorf("期望 ErrSemesterNotFound，实际: %v", err)
	}
}

// ── Delete 测试 ──

func TestSemesterService_Delete_Success(t *testing.T) {
	svc, semesterRepo := setupTestSemesterService()
	semesterRepo.semesters["sem-001"] = &model.Semester{
		SemesterID: "sem-001",
		Name:       "测试学期",
		StartDate:  time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
		EndDate:    time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	}

	err := svc.Delete(context.Background(), "sem-001", "admin-001")
	if err != nil {
		t.Fatalf("Delete 应成功: %v", err)
	}
}

func TestSemesterService_Delete_NotFound(t *testing.T) {
	svc, _ := setupTestSemesterService()

	err := svc.Delete(context.Background(), "nonexistent", "admin-001")
	if !errors.Is(err, ErrSemesterNotFound) {
		t.Errorf("期望 ErrSemesterNotFound，实际: %v", err)
	}
}

// ── Update 测试 ──

func TestSemesterService_Update_Success(t *testing.T) {
	svc, semesterRepo := setupTestSemesterService()
	semesterRepo.semesters["sem-001"] = &model.Semester{
		SemesterID:    "sem-001",
		Name:          "旧名称",
		StartDate:     time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		FirstWeekType: "odd",
		Status:        "active",
	}

	newName := "新名称"
	req := &dto.UpdateSemesterRequest{Name: &newName}

	result, err := svc.Update(context.Background(), "sem-001", req, "admin-001")
	if err != nil {
		t.Fatalf("Update 应成功: %v", err)
	}
	if result.Name != "新名称" {
		t.Errorf("期望Name=新名称，实际=%s", result.Name)
	}
}

func TestSemesterService_Update_NotFound(t *testing.T) {
	svc, _ := setupTestSemesterService()

	newName := "新名称"
	req := &dto.UpdateSemesterRequest{Name: &newName}

	_, err := svc.Update(context.Background(), "nonexistent", req, "admin-001")
	if !errors.Is(err, ErrSemesterNotFound) {
		t.Errorf("期望 ErrSemesterNotFound，实际: %v", err)
	}
}
