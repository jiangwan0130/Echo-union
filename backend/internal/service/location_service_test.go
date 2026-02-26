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

func setupTestLocationService() (LocationService, *mockLocationRepo) {
	locationRepo := newMockLocationRepo()
	repo := &repository.Repository{
		User:         newMockUserRepo(),
		Department:   newMockDeptRepo(),
		Semester:     newMockSemesterRepo(),
		TimeSlot:     newMockTimeSlotRepo(),
		Location:     locationRepo,
		SystemConfig: newMockSystemConfigRepo(),
		ScheduleRule: newMockScheduleRuleRepo(),
	}
	logger := zap.NewNop()
	svc := NewLocationService(repo, logger)
	return svc, locationRepo
}

// ── Create 测试 ──

func TestLocationService_Create_Success(t *testing.T) {
	svc, _ := setupTestLocationService()

	req := &dto.CreateLocationRequest{
		Name:      "学生会办公室",
		Address:   "行政楼3层",
		IsDefault: true,
	}

	result, err := svc.Create(context.Background(), req, "admin-001")
	if err != nil {
		t.Fatalf("Create 应成功: %v", err)
	}
	if result.Name != "学生会办公室" {
		t.Errorf("期望Name=学生会办公室，实际=%s", result.Name)
	}
	if !result.IsDefault {
		t.Error("期望IsDefault=true")
	}
}

// ── GetByID 测试 ──

func TestLocationService_GetByID_Success(t *testing.T) {
	svc, locRepo := setupTestLocationService()
	locRepo.locations["loc-001"] = &model.Location{
		LocationID: "loc-001",
		Name:       "测试地点",
		IsActive:   true,
	}

	result, err := svc.GetByID(context.Background(), "loc-001")
	if err != nil {
		t.Fatalf("GetByID 应成功: %v", err)
	}
	if result.Name != "测试地点" {
		t.Errorf("期望Name=测试地点，实际=%s", result.Name)
	}
}

func TestLocationService_GetByID_NotFound(t *testing.T) {
	svc, _ := setupTestLocationService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrLocationNotFound) {
		t.Errorf("期望 ErrLocationNotFound，实际: %v", err)
	}
}

// ── List 测试 ──

func TestLocationService_List_ActiveOnly(t *testing.T) {
	svc, locRepo := setupTestLocationService()
	locRepo.locations["loc-001"] = &model.Location{
		LocationID: "loc-001", Name: "活跃地点", IsActive: true,
	}
	locRepo.locations["loc-002"] = &model.Location{
		LocationID: "loc-002", Name: "停用地点", IsActive: false,
	}

	req := &dto.LocationListRequest{IncludeInactive: false}
	locations, err := svc.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}

	for _, l := range locations {
		if l.Name == "停用地点" {
			t.Error("不应返回停用地点")
		}
	}
}

func TestLocationService_List_IncludeInactive(t *testing.T) {
	svc, locRepo := setupTestLocationService()
	locRepo.locations["loc-001"] = &model.Location{
		LocationID: "loc-001", Name: "活跃地点", IsActive: true,
	}
	locRepo.locations["loc-002"] = &model.Location{
		LocationID: "loc-002", Name: "停用地点", IsActive: false,
	}

	req := &dto.LocationListRequest{IncludeInactive: true}
	locations, err := svc.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}

	if len(locations) < 2 {
		t.Errorf("期望至少2个地点，实际=%d", len(locations))
	}
}

// ── Update 测试 ──

func TestLocationService_Update_Success(t *testing.T) {
	svc, locRepo := setupTestLocationService()
	locRepo.locations["loc-001"] = &model.Location{
		LocationID: "loc-001",
		Name:       "旧名称",
		IsActive:   true,
	}

	newName := "新名称"
	req := &dto.UpdateLocationRequest{Name: &newName}

	result, err := svc.Update(context.Background(), "loc-001", req, "admin-001")
	if err != nil {
		t.Fatalf("Update 应成功: %v", err)
	}
	if result.Name != "新名称" {
		t.Errorf("期望Name=新名称，实际=%s", result.Name)
	}
}

func TestLocationService_Update_NotFound(t *testing.T) {
	svc, _ := setupTestLocationService()

	newName := "新名称"
	req := &dto.UpdateLocationRequest{Name: &newName}

	_, err := svc.Update(context.Background(), "nonexistent", req, "admin-001")
	if !errors.Is(err, ErrLocationNotFound) {
		t.Errorf("期望 ErrLocationNotFound，实际: %v", err)
	}
}

// ── Delete 测试 ──

func TestLocationService_Delete_Success(t *testing.T) {
	svc, locRepo := setupTestLocationService()
	locRepo.locations["loc-001"] = &model.Location{
		LocationID: "loc-001", Name: "测试", IsActive: true,
	}

	err := svc.Delete(context.Background(), "loc-001", "admin-001")
	if err != nil {
		t.Fatalf("Delete 应成功: %v", err)
	}
}

func TestLocationService_Delete_NotFound(t *testing.T) {
	svc, _ := setupTestLocationService()

	err := svc.Delete(context.Background(), "nonexistent", "admin-001")
	if !errors.Is(err, ErrLocationNotFound) {
		t.Errorf("期望 ErrLocationNotFound，实际: %v", err)
	}
}
