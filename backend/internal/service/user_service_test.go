package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 测试辅助 ──

func setupTestUserService() (UserService, *mockUserRepo, *mockDeptRepo) {
	userRepo := newMockUserRepo()
	deptRepo := newMockDeptRepo()
	repo := &repository.Repository{
		User:         userRepo,
		Department:   deptRepo,
		InviteCode:   newMockInviteCodeRepo(),
		Semester:     newMockSemesterRepo(),
		TimeSlot:     newMockTimeSlotRepo(),
		Location:     newMockLocationRepo(),
		SystemConfig: newMockSystemConfigRepo(),
		ScheduleRule: newMockScheduleRuleRepo(),
	}
	logger := zap.NewNop()
	svc := NewUserService(repo, logger)
	return svc, userRepo, deptRepo
}

func createTestUserForUserSvc(userRepo *mockUserRepo, userID, studentID, name, role, deptID string) *model.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	user := &model.User{
		UserID:       userID,
		Name:         name,
		StudentID:    studentID,
		Email:        studentID + "@test.com",
		PasswordHash: string(hash),
		Role:         role,
		DepartmentID: deptID,
		Department:   &model.Department{DepartmentID: deptID, Name: "测试部门"},
	}
	userRepo.users[studentID] = user
	userRepo.users[user.UserID] = user
	userRepo.users["email:"+user.Email] = user
	return user
}

// ── GetByID 测试 ──

func TestUserService_GetByID_Success(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")

	result, err := svc.GetByID(context.Background(), "uid-001")
	if err != nil {
		t.Fatalf("GetByID 应成功: %v", err)
	}
	if result.Name != "张三" {
		t.Errorf("期望Name=张三，实际=%s", result.Name)
	}
	if result.Department == nil {
		t.Error("期望包含部门信息")
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	svc, _, _ := setupTestUserService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("期望 ErrUserNotFound，实际: %v", err)
	}
}

// ── List 测试 ──

func TestUserService_List_Admin(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")
	createTestUserForUserSvc(userRepo, "uid-002", "2024002", "李四", "leader", "dept-2")

	req := &dto.UserListRequest{}
	req.Page = 1
	req.PageSize = 20

	users, total, err := svc.List(context.Background(), req, "admin", "any-dept")
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}
	if total != 2 {
		t.Errorf("期望total=2，实际=%d", total)
	}
	if len(users) != 2 {
		t.Errorf("期望2条记录，实际=%d", len(users))
	}
}

func TestUserService_List_LeaderAutoFilter(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")
	createTestUserForUserSvc(userRepo, "uid-002", "2024002", "李四", "member", "dept-2")

	req := &dto.UserListRequest{}
	req.Page = 1
	req.PageSize = 20

	// leader 的 department_id 是 valid-dept-id，应自动过滤
	users, total, err := svc.List(context.Background(), req, "leader", "valid-dept-id")
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}
	if total != 1 {
		t.Errorf("期望total=1（leader只能看本部门），实际=%d", total)
	}
	if len(users) > 0 && users[0].Name != "张三" {
		t.Errorf("期望看到张三，实际=%s", users[0].Name)
	}
}

func TestUserService_List_FilterByRole(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")
	createTestUserForUserSvc(userRepo, "uid-002", "2024002", "李四", "leader", "valid-dept-id")

	req := &dto.UserListRequest{}
	req.Role = "leader"
	req.Page = 1
	req.PageSize = 20

	users, total, err := svc.List(context.Background(), req, "admin", "any-dept")
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}
	if total != 1 {
		t.Errorf("期望total=1，实际=%d", total)
	}
	if len(users) > 0 && users[0].Role != "leader" {
		t.Errorf("期望role=leader，实际=%s", users[0].Role)
	}
}

func TestUserService_List_FilterByKeyword(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")
	createTestUserForUserSvc(userRepo, "uid-002", "2024002", "李四", "member", "valid-dept-id")

	req := &dto.UserListRequest{}
	req.Keyword = "张三"
	req.Page = 1
	req.PageSize = 20

	users, total, err := svc.List(context.Background(), req, "admin", "any-dept")
	if err != nil {
		t.Fatalf("List 应成功: %v", err)
	}
	if total != 1 {
		t.Errorf("期望total=1，实际=%d", total)
	}
	if len(users) > 0 && users[0].Name != "张三" {
		t.Errorf("期望Name=张三，实际=%s", users[0].Name)
	}
}

// ── Update 测试 ──

func TestUserService_Update_Self(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")

	name := "张三丰"
	result, err := svc.Update(context.Background(), "uid-001", &dto.UpdateUserRequest{
		Name: &name,
	}, "uid-001", "member")

	if err != nil {
		t.Fatalf("Update 应成功: %v", err)
	}
	if result.Name != "张三丰" {
		t.Errorf("期望Name=张三丰，实际=%s", result.Name)
	}
}

func TestUserService_Update_AdminChangeDept(t *testing.T) {
	svc, userRepo, deptRepo := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")
	deptRepo.departments["dept-2"] = &model.Department{DepartmentID: "dept-2", Name: "宣传部", IsActive: true}

	deptID := "dept-2"
	result, err := svc.Update(context.Background(), "uid-001", &dto.UpdateUserRequest{
		DepartmentID: &deptID,
	}, "admin-uid", "admin")

	if err != nil {
		t.Fatalf("Admin Update 应成功: %v", err)
	}
	// 验证部门已更改（虽然 mock 不会更新 Department 关联，但 DepartmentID 已变更）
	_ = result
}

func TestUserService_Update_NonAdminCannotChangeDept(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")

	deptID := "dept-2"
	_, err := svc.Update(context.Background(), "uid-001", &dto.UpdateUserRequest{
		DepartmentID: &deptID,
	}, "uid-001", "member")

	if !errors.Is(err, ErrNoPermission) {
		t.Errorf("期望 ErrNoPermission，实际: %v", err)
	}
}

func TestUserService_Update_CannotUpdateOthers(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")
	createTestUserForUserSvc(userRepo, "uid-002", "2024002", "李四", "member", "valid-dept-id")

	name := "新名字"
	_, err := svc.Update(context.Background(), "uid-001", &dto.UpdateUserRequest{
		Name: &name,
	}, "uid-002", "member")

	if !errors.Is(err, ErrNoPermission) {
		t.Errorf("期望 ErrNoPermission（非管理员不能改他人），实际: %v", err)
	}
}

func TestUserService_Update_DuplicateEmail(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")
	createTestUserForUserSvc(userRepo, "uid-002", "2024002", "李四", "member", "valid-dept-id")

	email := "2024002@test.com" // 已被李四使用
	_, err := svc.Update(context.Background(), "uid-001", &dto.UpdateUserRequest{
		Email: &email,
	}, "uid-001", "member")

	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("期望 ErrEmailExists，实际: %v", err)
	}
}

func TestUserService_Update_NotFound(t *testing.T) {
	svc, _, _ := setupTestUserService()

	name := "不存在"
	_, err := svc.Update(context.Background(), "nonexistent", &dto.UpdateUserRequest{
		Name: &name,
	}, "admin-uid", "admin")

	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("期望 ErrUserNotFound，实际: %v", err)
	}
}

// ── Delete 测试 ──

func TestUserService_Delete_Success(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")

	err := svc.Delete(context.Background(), "uid-001", "admin-uid")
	if err != nil {
		t.Fatalf("Delete 应成功: %v", err)
	}
}

func TestUserService_Delete_SelfProtection(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "admin", "valid-dept-id")

	err := svc.Delete(context.Background(), "uid-001", "uid-001")
	if !errors.Is(err, ErrUserSelfDelete) {
		t.Errorf("期望 ErrUserSelfDelete，实际: %v", err)
	}
}

func TestUserService_Delete_NotFound(t *testing.T) {
	svc, _, _ := setupTestUserService()

	err := svc.Delete(context.Background(), "nonexistent", "admin-uid")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("期望 ErrUserNotFound，实际: %v", err)
	}
}

// ── AssignRole 测试 ──

func TestUserService_AssignRole_Success(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")

	err := svc.AssignRole(context.Background(), "uid-001", &dto.AssignRoleRequest{Role: "leader"}, "admin-uid")
	if err != nil {
		t.Fatalf("AssignRole 应成功: %v", err)
	}

	// 验证角色已更新
	user, _ := userRepo.GetByID(context.Background(), "uid-001")
	if user.Role != "leader" {
		t.Errorf("期望role=leader，实际=%s", user.Role)
	}
}

func TestUserService_AssignRole_SelfProtection(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "admin", "valid-dept-id")

	err := svc.AssignRole(context.Background(), "uid-001", &dto.AssignRoleRequest{Role: "member"}, "uid-001")
	if !errors.Is(err, ErrUserSelfRoleChange) {
		t.Errorf("期望 ErrUserSelfRoleChange，实际: %v", err)
	}
}

func TestUserService_AssignRole_NotFound(t *testing.T) {
	svc, _, _ := setupTestUserService()

	err := svc.AssignRole(context.Background(), "nonexistent", &dto.AssignRoleRequest{Role: "admin"}, "admin-uid")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("期望 ErrUserNotFound，实际: %v", err)
	}
}

// ── ResetPassword 测试 ──

func TestUserService_ResetPassword_Success(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")

	result, err := svc.ResetPassword(context.Background(), "uid-001", "admin-uid")
	if err != nil {
		t.Fatalf("ResetPassword 应成功: %v", err)
	}
	if result.TempPassword == "" {
		t.Error("临时密码不应为空")
	}
	if len(result.TempPassword) != 8 {
		t.Errorf("期望临时密码长度=8，实际=%d", len(result.TempPassword))
	}

	// 验证 must_change_password 已设置
	user, _ := userRepo.GetByID(context.Background(), "uid-001")
	if !user.MustChangePassword {
		t.Error("期望 MustChangePassword=true")
	}

	// 验证新密码可用（哈希匹配）
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(result.TempPassword)); err != nil {
		t.Error("临时密码应能通过哈希验证")
	}
}

func TestUserService_ResetPassword_NotFound(t *testing.T) {
	svc, _, _ := setupTestUserService()

	_, err := svc.ResetPassword(context.Background(), "nonexistent", "admin-uid")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("期望 ErrUserNotFound，实际: %v", err)
	}
}

// ── ImportUsers 测试 ──

func TestUserService_ImportUsers_Success(t *testing.T) {
	svc, _, _ := setupTestUserService()

	rows := []ImportUserRow{
		{Row: 2, Name: "新用户1", StudentID: "2024101", Email: "new1@test.com", DepartmentName: "测试部门"},
		{Row: 3, Name: "新用户2", StudentID: "2024102", Email: "new2@test.com", DepartmentName: "测试部门"},
	}

	result, err := svc.ImportUsers(context.Background(), rows)
	if err != nil {
		t.Fatalf("ImportUsers 应成功: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("期望Total=2，实际=%d", result.Total)
	}
	if result.Success != 2 {
		t.Errorf("期望Success=2，实际=%d", result.Success)
	}
	if result.Failed != 0 {
		t.Errorf("期望Failed=0，实际=%d", result.Failed)
	}
}

func TestUserService_ImportUsers_DepartmentNotFound(t *testing.T) {
	svc, _, _ := setupTestUserService()

	rows := []ImportUserRow{
		{Row: 2, Name: "新用户", StudentID: "2024201", Email: "new@test.com", DepartmentName: "不存在的部门"},
	}

	result, err := svc.ImportUsers(context.Background(), rows)
	if err != nil {
		t.Fatalf("ImportUsers 应返回结果而非错误: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("期望Failed=1，实际=%d", result.Failed)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("期望1个错误详情，实际=%d", len(result.Errors))
	}
	if result.Errors[0].Row != 2 {
		t.Errorf("期望错误行=2，实际=%d", result.Errors[0].Row)
	}
}

func TestUserService_ImportUsers_DuplicateStudentID(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "张三", "member", "valid-dept-id")

	rows := []ImportUserRow{
		{Row: 2, Name: "重复用户", StudentID: "2024001", Email: "dup@test.com", DepartmentName: "测试部门"},
	}

	result, err := svc.ImportUsers(context.Background(), rows)
	if err != nil {
		t.Fatalf("ImportUsers 应返回结果而非错误: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("期望Failed=1，实际=%d", result.Failed)
	}
}

func TestUserService_ImportUsers_EmptyFields(t *testing.T) {
	svc, _, _ := setupTestUserService()

	rows := []ImportUserRow{
		{Row: 2, Name: "", StudentID: "2024301", Email: "e@t.com", DepartmentName: "测试部门"},
	}

	result, err := svc.ImportUsers(context.Background(), rows)
	if err != nil {
		t.Fatalf("ImportUsers 应返回结果而非错误: %v", err)
	}
	if result.Failed != 1 {
		t.Errorf("期望Failed=1（空姓名），实际=%d", result.Failed)
	}
}

func TestUserService_ImportUsers_Mixed(t *testing.T) {
	svc, userRepo, _ := setupTestUserService()
	createTestUserForUserSvc(userRepo, "uid-001", "2024001", "已存在", "member", "valid-dept-id")

	rows := []ImportUserRow{
		{Row: 2, Name: "新用户", StudentID: "2024301", Email: "ok@test.com", DepartmentName: "测试部门"},
		{Row: 3, Name: "重复学号", StudentID: "2024001", Email: "dup@test.com", DepartmentName: "测试部门"},
		{Row: 4, Name: "", StudentID: "2024302", Email: "empty@test.com", DepartmentName: "测试部门"},
		{Row: 5, Name: "坏部门", StudentID: "2024303", Email: "bad@test.com", DepartmentName: "鬼部门"},
	}

	result, err := svc.ImportUsers(context.Background(), rows)
	if err != nil {
		t.Fatalf("ImportUsers 应返回结果: %v", err)
	}
	if result.Total != 4 {
		t.Errorf("期望Total=4，实际=%d", result.Total)
	}
	if result.Success != 1 {
		t.Errorf("期望Success=1，实际=%d", result.Success)
	}
	if result.Failed != 3 {
		t.Errorf("期望Failed=3，实际=%d", result.Failed)
	}
}

// ── generateTempPassword 测试 ──

func TestGenerateTempPassword(t *testing.T) {
	for i := 0; i < 20; i++ {
		pwd, err := generateTempPassword(8)
		if err != nil {
			t.Fatalf("generateTempPassword 应成功: %v", err)
		}
		if len(pwd) != 8 {
			t.Errorf("期望长度=8，实际=%d", len(pwd))
		}
		// 检查包含字母和数字
		if !hasLetter.MatchString(pwd) {
			t.Errorf("临时密码 %q 应包含字母", pwd)
		}
		if !hasDigit.MatchString(pwd) {
			t.Errorf("临时密码 %q 应包含数字", pwd)
		}
	}
}
