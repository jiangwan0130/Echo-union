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

func setupTestTimeSlotService() (TimeSlotService, *mockTimeSlotRepo, *mockSemesterRepo) {
	timeSlotRepo := newMockTimeSlotRepo()
	semesterRepo := newMockSemesterRepo()
	repo := &repository.Repository{
		User:         newMockUserRepo(),
		Department:   newMockDeptRepo(),
		InviteCode:   newMockInviteCodeRepo(),
		Semester:     semesterRepo,
		TimeSlot:     timeSlotRepo,
		Location:     newMockLocationRepo(),
		SystemConfig: newMockSystemConfigRepo(),
		ScheduleRule: newMockScheduleRuleRepo(),
	}
	logger := zap.NewNop()
	svc := NewTimeSlotService(repo, logger)
	return svc, timeSlotRepo, semesterRepo
}

// ── Create 测试 ──

func TestTimeSlotService_Create_Success(t *testing.T) {
	svc, _, _ := setupTestTimeSlotService()

	req := &dto.CreateTimeSlotRequest{
		Name:      "周一上午第1节",
		StartTime: "08:10",
		EndTime:   "10:05",
		DayOfWeek: 1,
	}

	result, err := svc.Create(context.Background(), req, "admin-001")
	if err != nil {
		t.Fatalf("Create 应成功: %v", err)
	}
	if result.Name != "周一上午第1节" {
		t.Errorf("期望Name=周一上午第1节，实际=%s", result.Name)
	}
	if result.DayOfWeek != 1 {
		t.Errorf("期望DayOfWeek=1，实际=%d", result.DayOfWeek)
	}
}

func TestTimeSlotService_Create_WithSemester(t *testing.T) {
	svc, _, semesterRepo := setupTestTimeSlotService()
	semesterRepo.semesters["sem-001"] = &model.Semester{
		SemesterID: "sem-001",
		Name:       "测试学期",
	}

	semID := "sem-001"
	req := &dto.CreateTimeSlotRequest{
		Name:       "周一上午第1节",
		SemesterID: &semID,
		StartTime:  "08:10",
		EndTime:    "10:05",
		DayOfWeek:  1,
	}

	result, err := svc.Create(context.Background(), req, "admin-001")
	if err != nil {
		t.Fatalf("Create 应成功: %v", err)
	}
	if result.SemesterID == nil || *result.SemesterID != "sem-001" {
		t.Error("期望关联学期sem-001")
	}
}

func TestTimeSlotService_Create_SemesterNotFound(t *testing.T) {
	svc, _, _ := setupTestTimeSlotService()

	badSemID := "nonexistent"
	req := &dto.CreateTimeSlotRequest{
		Name:       "周一上午第1节",
		SemesterID: &badSemID,
		StartTime:  "08:10",
		EndTime:    "10:05",
		DayOfWeek:  1,
	}

	_, err := svc.Create(context.Background(), req, "admin-001")
	if !errors.Is(err, ErrSemesterNotFound) {
		t.Errorf("期望 ErrSemesterNotFound，实际: %v", err)
	}
}

// ── GetByID 测试 ──

func TestTimeSlotService_GetByID_Success(t *testing.T) {
	svc, tsRepo, _ := setupTestTimeSlotService()
	tsRepo.slots["ts-001"] = &model.TimeSlot{
		TimeSlotID: "ts-001",
		Name:       "周一上午第1节",
		StartTime:  "08:10",
		EndTime:    "10:05",
		DayOfWeek:  1,
		IsActive:   true,
	}

	result, err := svc.GetByID(context.Background(), "ts-001")
	if err != nil {
		t.Fatalf("GetByID 应成功: %v", err)
	}
	if result.Name != "周一上午第1节" {
		t.Errorf("期望Name=周一上午第1节，实际=%s", result.Name)
	}
}

func TestTimeSlotService_GetByID_NotFound(t *testing.T) {
	svc, _, _ := setupTestTimeSlotService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrTimeSlotNotFound) {
		t.Errorf("期望 ErrTimeSlotNotFound，实际: %v", err)
	}
}

// ── List 测试 ──

func TestTimeSlotService_List_FilterByDayOfWeek(t *testing.T) {
	svc, tsRepo, _ := setupTestTimeSlotService()
	tsRepo.slots["ts-001"] = &model.TimeSlot{
		TimeSlotID: "ts-001", Name: "周一", DayOfWeek: 1, IsActive: true,
	}
	tsRepo.slots["ts-002"] = &model.TimeSlot{
		TimeSlotID: "ts-002", Name: "周二", DayOfWeek: 2, IsActive: true,
	}

	day := 1
	req := &dto.TimeSlotListRequest{DayOfWeek: &day}
	slots, err := svc.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}

	for _, s := range slots {
		if s.DayOfWeek != 1 {
			t.Errorf("期望只返回周一的时间段，实际=%d", s.DayOfWeek)
		}
	}
}

// ── Update 测试 ──

func TestTimeSlotService_Update_Success(t *testing.T) {
	svc, tsRepo, _ := setupTestTimeSlotService()
	tsRepo.slots["ts-001"] = &model.TimeSlot{
		TimeSlotID: "ts-001",
		Name:       "旧名称",
		StartTime:  "08:10",
		EndTime:    "10:05",
		DayOfWeek:  1,
		IsActive:   true,
	}

	newName := "新名称"
	req := &dto.UpdateTimeSlotRequest{Name: &newName}

	result, err := svc.Update(context.Background(), "ts-001", req, "admin-001")
	if err != nil {
		t.Fatalf("Update 应成功: %v", err)
	}
	if result.Name != "新名称" {
		t.Errorf("期望Name=新名称，实际=%s", result.Name)
	}
}

// ── Delete 测试 ──

func TestTimeSlotService_Delete_Success(t *testing.T) {
	svc, tsRepo, _ := setupTestTimeSlotService()
	tsRepo.slots["ts-001"] = &model.TimeSlot{
		TimeSlotID: "ts-001", Name: "测试", IsActive: true,
	}

	err := svc.Delete(context.Background(), "ts-001", "admin-001")
	if err != nil {
		t.Fatalf("Delete 应成功: %v", err)
	}
}

func TestTimeSlotService_Delete_NotFound(t *testing.T) {
	svc, _, _ := setupTestTimeSlotService()

	err := svc.Delete(context.Background(), "nonexistent", "admin-001")
	if !errors.Is(err, ErrTimeSlotNotFound) {
		t.Errorf("期望 ErrTimeSlotNotFound，实际: %v", err)
	}
}
