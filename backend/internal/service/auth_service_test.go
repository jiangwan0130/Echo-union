package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"echo-union/backend/config"
	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
	"echo-union/backend/pkg/jwt"
)

// ── Mock Repositories ──

type mockUserRepo struct {
	users map[string]*model.User // key: student_id or user_id
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	if user.UserID == "" {
		user.UserID = "test-user-" + user.StudentID
	}
	m.users[user.StudentID] = user
	m.users[user.UserID] = user
	if user.Email != "" {
		m.users["email:"+user.Email] = user
	}
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id string) (*model.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) GetByStudentID(_ context.Context, studentID string) (*model.User, error) {
	if u, ok := m.users[studentID]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	// 先检查索引
	if u, ok := m.users["email:"+email]; ok {
		return u, nil
	}
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) Update(_ context.Context, user *model.User) error {
	m.users[user.StudentID] = user
	m.users[user.UserID] = user
	if user.Email != "" {
		m.users["email:"+user.Email] = user
	}
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id string, deletedBy string) error {
	// 找到并删除
	for key, u := range m.users {
		if u.UserID == id {
			delete(m.users, key)
		}
	}
	return nil
}

func (m *mockUserRepo) List(_ context.Context, offset, limit int) ([]model.User, int64, error) {
	return m.ListWithFilters(nil, nil, offset, limit)
}

func (m *mockUserRepo) ListWithFilters(_ context.Context, filters *repository.UserListFilters, offset, limit int) ([]model.User, int64, error) {
	// 去重收集所有用户
	seen := make(map[string]bool)
	var all []model.User
	for _, u := range m.users {
		if !seen[u.UserID] {
			seen[u.UserID] = true
			match := true
			if filters != nil {
				if filters.DepartmentID != "" && u.DepartmentID != filters.DepartmentID {
					match = false
				}
				if filters.Role != "" && u.Role != filters.Role {
					match = false
				}
				if filters.Keyword != "" {
					// 简单包含匹配
					kw := filters.Keyword
					if !(contains(u.Name, kw) || contains(u.StudentID, kw)) {
						match = false
					}
				}
			}
			if match {
				all = append(all, *u)
			}
		}
	}
	total := int64(len(all))
	// 简单分页
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	if offset > len(all) {
		return nil, total, nil
	}
	return all[offset:end], total, nil
}

func (m *mockUserRepo) BatchCreate(_ context.Context, users []*model.User) (int, error) {
	for _, u := range users {
		_ = m.Create(nil, u)
	}
	return len(users), nil
}

func (m *mockUserRepo) ListByIDs(_ context.Context, ids []string) ([]model.User, error) {
	var result []model.User
	for _, id := range ids {
		if u, ok := m.users[id]; ok {
			result = append(result, *u)
		}
	}
	return result, nil
}

// contains 简单字符串包含检查（用于 mock 关键词搜索）
func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (s == sub || findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

type mockDeptRepo struct {
	departments  map[string]*model.Department
	memberCounts map[string]int64
}

func newMockDeptRepo() *mockDeptRepo {
	return &mockDeptRepo{
		departments: map[string]*model.Department{
			"valid-dept-id": {DepartmentID: "valid-dept-id", Name: "测试部门", IsActive: true},
		},
		memberCounts: make(map[string]int64),
	}
}

func (m *mockDeptRepo) Create(_ context.Context, dept *model.Department) error {
	m.departments[dept.DepartmentID] = dept
	return nil
}
func (m *mockDeptRepo) GetByID(_ context.Context, id string) (*model.Department, error) {
	if d, ok := m.departments[id]; ok {
		return d, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockDeptRepo) List(_ context.Context) ([]model.Department, error) {
	var result []model.Department
	for _, d := range m.departments {
		if d.IsActive {
			result = append(result, *d)
		}
	}
	return result, nil
}
func (m *mockDeptRepo) Update(_ context.Context, dept *model.Department) error {
	m.departments[dept.DepartmentID] = dept
	return nil
}
func (m *mockDeptRepo) GetByName(_ context.Context, name string) (*model.Department, error) {
	for _, d := range m.departments {
		if d.Name == name {
			return d, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockDeptRepo) ListAll(_ context.Context) ([]model.Department, error) {
	var result []model.Department
	for _, d := range m.departments {
		result = append(result, *d)
	}
	return result, nil
}
func (m *mockDeptRepo) Delete(_ context.Context, id string, deletedBy string) error {
	delete(m.departments, id)
	return nil
}
func (m *mockDeptRepo) CountMembers(_ context.Context, departmentID string) (int64, error) {
	if count, ok := m.memberCounts[departmentID]; ok {
		return count, nil
	}
	return 0, nil
}

func (m *mockDeptRepo) BatchCountMembers(_ context.Context, departmentIDs []string) (map[string]int64, error) {
	result := make(map[string]int64, len(departmentIDs))
	for _, id := range departmentIDs {
		if count, ok := m.memberCounts[id]; ok {
			result[id] = count
		}
	}
	return result, nil
}

type mockInviteCodeRepo struct {
	codes map[string]*model.InviteCode
}

func newMockInviteCodeRepo() *mockInviteCodeRepo {
	return &mockInviteCodeRepo{codes: make(map[string]*model.InviteCode)}
}

func (m *mockInviteCodeRepo) Create(_ context.Context, code *model.InviteCode) error {
	if code.InviteCodeID == "" {
		code.InviteCodeID = "invite-" + code.Code
	}
	m.codes[code.Code] = code
	return nil
}

func (m *mockInviteCodeRepo) GetByCode(_ context.Context, code string) (*model.InviteCode, error) {
	if c, ok := m.codes[code]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockInviteCodeRepo) GetByCodeForUpdate(_ context.Context, code string) (*model.InviteCode, error) {
	// 在 mock 中与 GetByCode 行为一致
	return m.GetByCode(nil, code)
}

func (m *mockInviteCodeRepo) MarkUsed(_ context.Context, inviteCodeID, userID string) error {
	for _, c := range m.codes {
		if c.InviteCodeID == inviteCodeID {
			now := time.Now()
			c.UsedAt = &now
			c.UsedBy = &userID
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

// ── 测试辅助 ──

func setupTestAuthService() (AuthService, *mockUserRepo, *mockInviteCodeRepo) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			BaseURL: "http://localhost:8080",
		},
		Auth: config.AuthConfig{
			JWTSecret:               "test-secret-key-for-unit-testing-2026",
			AccessTokenTTL:          15 * time.Minute,
			RefreshTokenTTLDefault:  24 * time.Hour,
			RefreshTokenTTLRemember: 7 * 24 * time.Hour,
		},
	}

	userRepo := newMockUserRepo()
	inviteRepo := newMockInviteCodeRepo()
	deptRepo := newMockDeptRepo()
	repo := &repository.Repository{
		User:         userRepo,
		Department:   deptRepo,
		InviteCode:   inviteRepo,
		Semester:     newMockSemesterRepo(),
		TimeSlot:     newMockTimeSlotRepo(),
		Location:     newMockLocationRepo(),
		SystemConfig: newMockSystemConfigRepo(),
		ScheduleRule: newMockScheduleRuleRepo(),
	}

	jwtMgr := jwt.NewManager(&cfg.Auth)
	logger := zap.NewNop()

	svc := NewAuthService(cfg, repo, jwtMgr, nil, logger)
	return svc, userRepo, inviteRepo
}

func createTestUser(userRepo *mockUserRepo, studentID, password string) *model.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	user := &model.User{
		UserID:       "user-" + studentID,
		Name:         "测试用户",
		StudentID:    studentID,
		Email:        studentID + "@test.com",
		PasswordHash: string(hash),
		Role:         "member",
		DepartmentID: "valid-dept-id",
		Department:   &model.Department{DepartmentID: "valid-dept-id", Name: "测试部门"},
	}
	userRepo.users[studentID] = user
	userRepo.users[user.UserID] = user
	return user
}

// ── 登录测试 ──

func TestLogin_Success(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	result, err := svc.Login(context.Background(), &dto.LoginRequest{
		StudentID: "2024001",
		Password:  "password123",
	})

	if err != nil {
		t.Fatalf("Login 应成功，但返回错误: %v", err)
	}
	if result.AccessToken == "" {
		t.Error("AccessToken 不应为空")
	}
	if result.RefreshToken == "" {
		t.Error("RefreshToken 不应为空")
	}
	if result.User.StudentID != "2024001" {
		t.Errorf("期望 StudentID=2024001，实际=%s", result.User.StudentID)
	}
	if result.ExpiresIn != 900 {
		t.Errorf("期望 ExpiresIn=900，实际=%d", result.ExpiresIn)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		StudentID: "2024001",
		Password:  "wrong_password",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("期望 ErrInvalidCredentials，实际: %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, _, _ := setupTestAuthService()

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		StudentID: "nonexistent",
		Password:  "password123",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("期望 ErrInvalidCredentials，实际: %v", err)
	}
}

func TestLogin_RememberMe(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	result, err := svc.Login(context.Background(), &dto.LoginRequest{
		StudentID:  "2024001",
		Password:   "password123",
		RememberMe: true,
	})

	if err != nil {
		t.Fatalf("Login(RememberMe) 应成功: %v", err)
	}
	if result.RefreshToken == "" {
		t.Error("RefreshToken 不应为空")
	}
}

// ── 注册测试 ──

func TestRegister_Success(t *testing.T) {
	svc, _, inviteRepo := setupTestAuthService()

	// 预设有效邀请码
	inviteRepo.codes["TESTCODE1"] = &model.InviteCode{
		InviteCodeID: "invite-1",
		Code:         "TESTCODE1",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	result, err := svc.Register(context.Background(), &dto.RegisterRequest{
		InviteCode:   "TESTCODE1",
		Name:         "新用户",
		StudentID:    "2024099",
		Email:        "new@test.com",
		Password:     "password123",
		DepartmentID: "valid-dept-id",
	})

	if err != nil {
		t.Fatalf("Register 应成功: %v", err)
	}
	if result.Name != "新用户" {
		t.Errorf("期望Name=新用户，实际=%s", result.Name)
	}
	if result.Email != "new@test.com" {
		t.Errorf("期望Email=new@test.com，实际=%s", result.Email)
	}
}

func TestRegister_InvalidInviteCode(t *testing.T) {
	svc, _, _ := setupTestAuthService()

	_, err := svc.Register(context.Background(), &dto.RegisterRequest{
		InviteCode:   "INVALID",
		Name:         "新用户",
		StudentID:    "2024099",
		Email:        "new@test.com",
		Password:     "password123",
		DepartmentID: "valid-dept-id",
	})

	if !errors.Is(err, ErrInviteCodeInvalid) {
		t.Errorf("期望 ErrInviteCodeInvalid，实际: %v", err)
	}
}

func TestRegister_ExpiredInviteCode(t *testing.T) {
	svc, _, inviteRepo := setupTestAuthService()

	inviteRepo.codes["EXPIRED1"] = &model.InviteCode{
		InviteCodeID: "invite-expired",
		Code:         "EXPIRED1",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // 已过期
	}

	_, err := svc.Register(context.Background(), &dto.RegisterRequest{
		InviteCode:   "EXPIRED1",
		Name:         "新用户",
		StudentID:    "2024099",
		Email:        "new@test.com",
		Password:     "password123",
		DepartmentID: "valid-dept-id",
	})

	if !errors.Is(err, ErrInviteCodeInvalid) {
		t.Errorf("期望 ErrInviteCodeInvalid，实际: %v", err)
	}
}

func TestRegister_DuplicateStudentID(t *testing.T) {
	svc, userRepo, inviteRepo := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	inviteRepo.codes["CODE2"] = &model.InviteCode{
		InviteCodeID: "invite-2",
		Code:         "CODE2",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	_, err := svc.Register(context.Background(), &dto.RegisterRequest{
		InviteCode:   "CODE2",
		Name:         "重复用户",
		StudentID:    "2024001", // 已存在
		Email:        "dup@test.com",
		Password:     "password123",
		DepartmentID: "valid-dept-id",
	})

	if !errors.Is(err, ErrStudentIDExists) {
		t.Errorf("期望 ErrStudentIDExists，实际: %v", err)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	svc, _, inviteRepo := setupTestAuthService()

	inviteRepo.codes["CODE3"] = &model.InviteCode{
		InviteCodeID: "invite-3",
		Code:         "CODE3",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name     string
		password string
	}{
		{"仅数字", "12345678"},
		{"仅字母", "abcdefgh"},
		{"太短", "abc1"},
		{"太长", "abcdefghijklmnopqrst1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), &dto.RegisterRequest{
				InviteCode:   "CODE3",
				Name:         "测试",
				StudentID:    "20240" + tt.name,
				Email:        tt.name + "@test.com",
				Password:     tt.password,
				DepartmentID: "valid-dept-id",
			})
			if !errors.Is(err, ErrWeakPassword) {
				t.Errorf("密码 %q 期望 ErrWeakPassword，实际: %v", tt.password, err)
			}
		})
	}
}

// ── RefreshToken 测试 ──

func TestRefreshToken_Success(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	user := createTestUser(userRepo, "2024001", "password123")

	// 先登录获取 refresh token
	loginResult, err := svc.Login(context.Background(), &dto.LoginRequest{
		StudentID: "2024001",
		Password:  "password123",
	})
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}

	// 使用 refresh token 刷新
	result, err := svc.RefreshToken(context.Background(), loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken 应成功: %v", err)
	}

	if result.AccessToken == "" {
		t.Error("新 AccessToken 不应为空")
	}
	if result.User.StudentID != user.StudentID {
		t.Errorf("期望 StudentID=%s，实际=%s", user.StudentID, result.User.StudentID)
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc, _, _ := setupTestAuthService()

	_, err := svc.RefreshToken(context.Background(), "invalid.token.string")
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("期望 ErrTokenInvalid，实际: %v", err)
	}
}

func TestRefreshToken_AccessTokenNotAllowed(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	loginResult, _ := svc.Login(context.Background(), &dto.LoginRequest{
		StudentID: "2024001",
		Password:  "password123",
	})

	// 使用 access token 尝试刷新（应拒绝）
	_, err := svc.RefreshToken(context.Background(), loginResult.AccessToken)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("期望 ErrTokenInvalid（access token 不能用于刷新），实际: %v", err)
	}
}

// ── GenerateInvite 测试 ──

func TestGenerateInvite_Success(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	result, err := svc.GenerateInvite(context.Background(), "user-2024001", 7)
	if err != nil {
		t.Fatalf("GenerateInvite 应成功: %v", err)
	}

	if result.InviteCode == "" {
		t.Error("InviteCode 不应为空")
	}
	if len(result.InviteCode) != 9 {
		t.Errorf("邀请码长度期望 9，实际=%d", len(result.InviteCode))
	}
	if result.InviteURL == "" {
		t.Error("InviteURL 不应为空")
	}
}

func TestGenerateInvite_DefaultDays(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	result, err := svc.GenerateInvite(context.Background(), "user-2024001", 0)
	if err != nil {
		t.Fatalf("GenerateInvite(默认天数) 应成功: %v", err)
	}

	if result.ExpiresAt == "" {
		t.Error("ExpiresAt 不应为空")
	}
}

// ── ValidateInvite 测试 ──

func TestValidateInvite_Valid(t *testing.T) {
	svc, userRepo, inviteRepo := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	inviteRepo.codes["VALIDCODE"] = &model.InviteCode{
		InviteCodeID: "invite-valid",
		Code:         "VALIDCODE",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	result, err := svc.ValidateInvite(context.Background(), "VALIDCODE")
	if err != nil {
		t.Fatalf("ValidateInvite 应成功: %v", err)
	}
	if !result.Valid {
		t.Error("期望 Valid=true")
	}
}

func TestValidateInvite_Expired(t *testing.T) {
	svc, _, inviteRepo := setupTestAuthService()

	inviteRepo.codes["EXPCODE"] = &model.InviteCode{
		InviteCodeID: "invite-exp",
		Code:         "EXPCODE",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	}

	_, err := svc.ValidateInvite(context.Background(), "EXPCODE")
	if !errors.Is(err, ErrInviteCodeInvalid) {
		t.Errorf("期望 ErrInviteCodeInvalid，实际: %v", err)
	}
}

// ── ChangePassword 测试 ──

func TestChangePassword_Success(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	err := svc.ChangePassword(context.Background(), "user-2024001", &dto.ChangePasswordRequest{
		OldPassword: "password123",
		NewPassword: "newpass456",
	})

	if err != nil {
		t.Fatalf("ChangePassword 应成功: %v", err)
	}

	// 验证新密码可以登录
	_, err = svc.Login(context.Background(), &dto.LoginRequest{
		StudentID: "2024001",
		Password:  "newpass456",
	})
	if err != nil {
		t.Fatalf("修改密码后应能用新密码登录: %v", err)
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	err := svc.ChangePassword(context.Background(), "user-2024001", &dto.ChangePasswordRequest{
		OldPassword: "wrong_old",
		NewPassword: "newpass456",
	})

	if !errors.Is(err, ErrOldPasswordWrong) {
		t.Errorf("期望 ErrOldPasswordWrong，实际: %v", err)
	}
}

func TestChangePassword_WeakNewPassword(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	err := svc.ChangePassword(context.Background(), "user-2024001", &dto.ChangePasswordRequest{
		OldPassword: "password123",
		NewPassword: "12345678", // 仅数字
	})

	if !errors.Is(err, ErrWeakPassword) {
		t.Errorf("期望 ErrWeakPassword，实际: %v", err)
	}
}

// ── GetCurrentUser 测试 ──

func TestGetCurrentUser_Success(t *testing.T) {
	svc, userRepo, _ := setupTestAuthService()
	createTestUser(userRepo, "2024001", "password123")

	result, err := svc.GetCurrentUser(context.Background(), "user-2024001")
	if err != nil {
		t.Fatalf("GetCurrentUser 应成功: %v", err)
	}

	if result.StudentID != "2024001" {
		t.Errorf("期望 StudentID=2024001，实际=%s", result.StudentID)
	}
	if result.Department == nil || result.Department.Name != "测试部门" {
		t.Error("期望包含部门信息")
	}
}

func TestGetCurrentUser_NotFound(t *testing.T) {
	svc, _, _ := setupTestAuthService()

	_, err := svc.GetCurrentUser(context.Background(), "nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("期望 ErrUserNotFound，实际: %v", err)
	}
}
