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

func setupTestDepartmentService() (DepartmentService, *mockDeptRepo, *mockUserRepo, *mockSemesterRepo, *mockUserSemesterAssignmentRepo) {
	deptRepo := newMockDeptRepo()
	userRepo := newMockUserRepo()
	semRepo := newMockSemesterRepo()
	usaRepo := newMockUserSemesterAssignmentRepo()
	repo := &repository.Repository{
		User:                   userRepo,
		Department:             deptRepo,
		Semester:               semRepo,
		TimeSlot:               newMockTimeSlotRepo(),
		Location:               newMockLocationRepo(),
		SystemConfig:           newMockSystemConfigRepo(),
		ScheduleRule:           newMockScheduleRuleRepo(),
		UserSemesterAssignment: usaRepo,
	}
	logger := zap.NewNop()
	svc := NewDepartmentService(repo, logger)
	return svc, deptRepo, userRepo, semRepo, usaRepo
}

// ── Create 测试 ──

func TestDepartmentService_Create_Success(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	req := &dto.CreateDepartmentRequest{
		Name:        "宣传部",
		Description: "负责宣传工作",
	}

	result, err := svc.Create(context.Background(), req, "admin-001")
	if err != nil {
		t.Fatalf("Create 应成功: %v", err)
	}
	if result.Name != "宣传部" {
		t.Errorf("期望Name=宣传部，实际=%s", result.Name)
	}
	if result.Description != "负责宣传工作" {
		t.Errorf("期望Description=负责宣传工作，实际=%s", result.Description)
	}
	if !result.IsActive {
		t.Error("期望默认IsActive=true")
	}
}

func TestDepartmentService_Create_NameExists(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	// "测试部门" 已在 mockDeptRepo 初始化时存在
	req := &dto.CreateDepartmentRequest{
		Name: "测试部门",
	}

	_, err := svc.Create(context.Background(), req, "admin-001")
	if !errors.Is(err, ErrDepartmentNameExists) {
		t.Errorf("期望 ErrDepartmentNameExists，实际: %v", err)
	}
}

// ── GetByID 测试 ──

func TestDepartmentService_GetByID_Success(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	result, err := svc.GetByID(context.Background(), "valid-dept-id")
	if err != nil {
		t.Fatalf("GetByID 应成功: %v", err)
	}
	if result.Name != "测试部门" {
		t.Errorf("期望Name=测试部门，实际=%s", result.Name)
	}
}

func TestDepartmentService_GetByID_NotFound(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("期望 ErrDepartmentNotFound，实际: %v", err)
	}
}

// ── List 测试 ──

func TestDepartmentService_List_ActiveOnly(t *testing.T) {
	svc, deptRepo, _, _, _ := setupTestDepartmentService()
	deptRepo.departments["inactive-dept"] = &model.Department{
		DepartmentID: "inactive-dept",
		Name:         "停用部门",
		IsActive:     false,
	}

	req := &dto.DepartmentListRequest{IncludeInactive: false}
	depts, err := svc.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}

	// 只应返回活跃部门
	for _, d := range depts {
		if d.Name == "停用部门" {
			t.Error("不应返回停用部门")
		}
	}
}

func TestDepartmentService_List_IncludeInactive(t *testing.T) {
	svc, deptRepo, _, _, _ := setupTestDepartmentService()
	deptRepo.departments["inactive-dept"] = &model.Department{
		DepartmentID: "inactive-dept",
		Name:         "停用部门",
		IsActive:     false,
	}

	req := &dto.DepartmentListRequest{IncludeInactive: true}
	depts, err := svc.List(context.Background(), req)
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}

	if len(depts) < 2 {
		t.Errorf("期望至少2个部门，实际=%d", len(depts))
	}
}

// ── Update 测试 ──

func TestDepartmentService_Update_Success(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	newName := "新名称"
	newDesc := "新描述"
	req := &dto.UpdateDepartmentRequest{
		Name:        &newName,
		Description: &newDesc,
	}

	result, err := svc.Update(context.Background(), "valid-dept-id", req, "admin-001")
	if err != nil {
		t.Fatalf("Update 应成功: %v", err)
	}
	if result.Name != "新名称" {
		t.Errorf("期望Name=新名称，实际=%s", result.Name)
	}
	if result.Description != "新描述" {
		t.Errorf("期望Description=新描述，实际=%s", result.Description)
	}
}

func TestDepartmentService_Update_NotFound(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	newName := "新名称"
	req := &dto.UpdateDepartmentRequest{Name: &newName}

	_, err := svc.Update(context.Background(), "nonexistent", req, "admin-001")
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("期望 ErrDepartmentNotFound，实际: %v", err)
	}
}

func TestDepartmentService_Update_NameConflict(t *testing.T) {
	svc, deptRepo, _, _, _ := setupTestDepartmentService()
	deptRepo.departments["dept-2"] = &model.Department{
		DepartmentID: "dept-2",
		Name:         "其他部门",
		IsActive:     true,
	}

	// 尝试将 dept-2 改名为已存在的 "测试部门"
	existName := "测试部门"
	req := &dto.UpdateDepartmentRequest{Name: &existName}

	_, err := svc.Update(context.Background(), "dept-2", req, "admin-001")
	if !errors.Is(err, ErrDepartmentNameExists) {
		t.Errorf("期望 ErrDepartmentNameExists，实际: %v", err)
	}
}

// ── Delete 测试 ──

func TestDepartmentService_Delete_Success(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	err := svc.Delete(context.Background(), "valid-dept-id", "admin-001")
	if err != nil {
		t.Fatalf("Delete 应成功: %v", err)
	}
}

func TestDepartmentService_Delete_NotFound(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	err := svc.Delete(context.Background(), "nonexistent", "admin-001")
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("期望 ErrDepartmentNotFound，实际: %v", err)
	}
}

func TestDepartmentService_Delete_HasMembers(t *testing.T) {
	svc, deptRepo, _, _, _ := setupTestDepartmentService()

	// 模拟 CountMembers 返回 > 0
	deptRepo.memberCounts = map[string]int64{
		"valid-dept-id": 5,
	}

	err := svc.Delete(context.Background(), "valid-dept-id", "admin-001")
	if !errors.Is(err, ErrDepartmentHasMembers) {
		t.Errorf("期望 ErrDepartmentHasMembers，实际: %v", err)
	}
}

// ══════════════════════════════════════════════════════════
// GetMembers 测试
// ══════════════════════════════════════════════════════════

func TestDepartmentService_GetMembers_Success(t *testing.T) {
	svc, _, userRepo, _, _ := setupTestDepartmentService()

	// 准备部门成员
	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-1",
		StudentID:    "STU001",
		Name:         "张三",
		Email:        "zhangsan@test.com",
		Role:         "member",
		DepartmentID: "valid-dept-id",
	})
	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-2",
		StudentID:    "STU002",
		Name:         "李四",
		Email:        "lisi@test.com",
		Role:         "member",
		DepartmentID: "valid-dept-id",
	})

	result, err := svc.GetMembers(context.Background(), "valid-dept-id", "")
	if err != nil {
		t.Fatalf("GetMembers 应成功: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("期望2个成员，实际=%d", len(result))
	}
	// 无学期 ID 时 duty_required 默认 false
	for _, m := range result {
		if m.DutyRequired {
			t.Errorf("无学期ID时 DutyRequired 应为 false")
		}
		if m.TimetableStatus != "not_submitted" {
			t.Errorf("无学期ID时 TimetableStatus 应为 not_submitted，实际=%s", m.TimetableStatus)
		}
	}
}

func TestDepartmentService_GetMembers_WithSemester(t *testing.T) {
	svc, _, userRepo, semRepo, usaRepo := setupTestDepartmentService()

	// 准备学期
	_ = semRepo.Create(context.Background(), &model.Semester{
		SemesterID: "sem-1",
		Name:       "2024秋",
		IsActive:   true,
	})

	// 准备成员
	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-1",
		StudentID:    "STU001",
		Name:         "张三",
		Role:         "member",
		DepartmentID: "valid-dept-id",
	})
	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-2",
		StudentID:    "STU002",
		Name:         "李四",
		Role:         "member",
		DepartmentID: "valid-dept-id",
	})

	// 给 user-1 设置值班标记
	usaRepo.assignments = append(usaRepo.assignments, model.UserSemesterAssignment{
		AssignmentID:    "assign-1",
		UserID:          "user-1",
		SemesterID:      "sem-1",
		DutyRequired:    true,
		TimetableStatus: "submitted",
		User:            &model.User{UserID: "user-1", DepartmentID: "valid-dept-id"},
	})

	result, err := svc.GetMembers(context.Background(), "valid-dept-id", "sem-1")
	if err != nil {
		t.Fatalf("GetMembers 应成功: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("期望2个成员，实际=%d", len(result))
	}

	// 检查 user-1 的值班状态
	for _, m := range result {
		if m.UserID == "user-1" {
			if !m.DutyRequired {
				t.Error("user-1 DutyRequired 应为 true")
			}
			if m.TimetableStatus != "submitted" {
				t.Errorf("user-1 TimetableStatus 应为 submitted，实际=%s", m.TimetableStatus)
			}
		}
		if m.UserID == "user-2" {
			if m.DutyRequired {
				t.Error("user-2 DutyRequired 应为 false")
			}
		}
	}
}

func TestDepartmentService_GetMembers_DepartmentNotFound(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	_, err := svc.GetMembers(context.Background(), "nonexistent", "")
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("期望 ErrDepartmentNotFound，实际: %v", err)
	}
}

// ══════════════════════════════════════════════════════════
// SetDutyMembers 测试
// ══════════════════════════════════════════════════════════

func TestDepartmentService_SetDutyMembers_Success(t *testing.T) {
	svc, _, userRepo, semRepo, usaRepo := setupTestDepartmentService()

	// 准备学期
	_ = semRepo.Create(context.Background(), &model.Semester{
		SemesterID: "sem-1",
		Name:       "2024秋",
		IsActive:   true,
	})

	// 准备成员
	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-1",
		StudentID:    "STU001",
		Name:         "张三",
		Role:         "member",
		DepartmentID: "valid-dept-id",
	})
	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-2",
		StudentID:    "STU002",
		Name:         "李四",
		Role:         "member",
		DepartmentID: "valid-dept-id",
	})

	req := &dto.SetDutyMembersRequest{
		SemesterID: "sem-1",
		UserIDs:    []string{"user-1", "user-2"},
	}

	result, err := svc.SetDutyMembers(context.Background(), "valid-dept-id", req, "admin-001")
	if err != nil {
		t.Fatalf("SetDutyMembers 应成功: %v", err)
	}
	if result.TotalSet != 2 {
		t.Errorf("期望 TotalSet=2，实际=%d", result.TotalSet)
	}
	if result.DepartmentID != "valid-dept-id" {
		t.Errorf("期望 DepartmentID=valid-dept-id，实际=%s", result.DepartmentID)
	}

	// 检查 assignment 已创建
	dutyCount := 0
	for _, a := range usaRepo.assignments {
		if a.SemesterID == "sem-1" && a.DutyRequired {
			dutyCount++
		}
	}
	if dutyCount != 2 {
		t.Errorf("期望2条 DutyRequired=true 的记录，实际=%d", dutyCount)
	}
}

func TestDepartmentService_SetDutyMembers_DeptNotFound(t *testing.T) {
	svc, _, _, semRepo, _ := setupTestDepartmentService()

	_ = semRepo.Create(context.Background(), &model.Semester{
		SemesterID: "sem-1",
		Name:       "2024秋",
	})

	req := &dto.SetDutyMembersRequest{
		SemesterID: "sem-1",
		UserIDs:    []string{"user-1"},
	}

	_, err := svc.SetDutyMembers(context.Background(), "nonexistent", req, "admin-001")
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("期望 ErrDepartmentNotFound，实际: %v", err)
	}
}

func TestDepartmentService_SetDutyMembers_SemesterNotFound(t *testing.T) {
	svc, _, _, _, _ := setupTestDepartmentService()

	req := &dto.SetDutyMembersRequest{
		SemesterID: "nonexistent-sem",
		UserIDs:    []string{"user-1"},
	}

	_, err := svc.SetDutyMembers(context.Background(), "valid-dept-id", req, "admin-001")
	if !errors.Is(err, ErrSemesterNotFound) {
		t.Errorf("期望 ErrSemesterNotFound，实际: %v", err)
	}
}

func TestDepartmentService_SetDutyMembers_UserNotInDept(t *testing.T) {
	svc, deptRepo, userRepo, semRepo, _ := setupTestDepartmentService()

	_ = semRepo.Create(context.Background(), &model.Semester{
		SemesterID: "sem-1",
		Name:       "2024秋",
	})

	// 创建另一个部门和该部门的用户
	deptRepo.departments["other-dept"] = &model.Department{
		DepartmentID: "other-dept",
		Name:         "其他部门",
		IsActive:     true,
	}
	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-other",
		StudentID:    "STU999",
		Name:         "王五",
		Role:         "member",
		DepartmentID: "other-dept",
	})

	req := &dto.SetDutyMembersRequest{
		SemesterID: "sem-1",
		UserIDs:    []string{"user-other"},
	}

	_, err := svc.SetDutyMembers(context.Background(), "valid-dept-id", req, "admin-001")
	if !errors.Is(err, ErrDutyMemberNotInDepartment) {
		t.Errorf("期望 ErrDutyMemberNotInDepartment，实际: %v", err)
	}
}

func TestDepartmentService_SetDutyMembers_ClearExisting(t *testing.T) {
	svc, _, userRepo, semRepo, usaRepo := setupTestDepartmentService()

	_ = semRepo.Create(context.Background(), &model.Semester{
		SemesterID: "sem-1",
		Name:       "2024秋",
	})

	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-1",
		StudentID:    "STU001",
		Name:         "张三",
		Role:         "member",
		DepartmentID: "valid-dept-id",
	})
	_ = userRepo.Create(context.Background(), &model.User{
		UserID:       "user-2",
		StudentID:    "STU002",
		Name:         "李四",
		Role:         "member",
		DepartmentID: "valid-dept-id",
	})

	// 先设置 user-1 为值班人员（模拟已有记录）
	usaRepo.assignments = append(usaRepo.assignments, model.UserSemesterAssignment{
		AssignmentID: "old-assign",
		UserID:       "user-1",
		SemesterID:   "sem-1",
		DutyRequired: true,
		User:         &model.User{UserID: "user-1", DepartmentID: "valid-dept-id"},
	})

	// 现在只设置 user-2（意味着 user-1 应被清除）
	req := &dto.SetDutyMembersRequest{
		SemesterID: "sem-1",
		UserIDs:    []string{"user-2"},
	}

	result, err := svc.SetDutyMembers(context.Background(), "valid-dept-id", req, "admin-001")
	if err != nil {
		t.Fatalf("SetDutyMembers 应成功: %v", err)
	}
	if result.TotalSet != 1 {
		t.Errorf("期望 TotalSet=1，实际=%d", result.TotalSet)
	}

	// 验证 old-assign 的 DutyRequired 被清除
	for _, a := range usaRepo.assignments {
		if a.AssignmentID == "old-assign" && a.DutyRequired {
			t.Error("旧的 assignment DutyRequired 应被清除为 false")
		}
	}
}
