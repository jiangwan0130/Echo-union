package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 用户模块业务错误 ──

var (
	ErrUserSelfRoleChange = errors.New("不能修改自己的角色")
	ErrUserSelfDelete     = errors.New("不能删除自己")
	ErrDepartmentNotFound = errors.New("部门不存在")
	ErrNoPermission       = errors.New("无权操作")
)

// UserService 用户业务接口
type UserService interface {
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
	List(ctx context.Context, req *dto.UserListRequest, callerRole, callerDeptID string) ([]dto.UserResponse, int64, error)
	Update(ctx context.Context, id string, req *dto.UpdateUserRequest, callerID, callerRole string) (*dto.UserResponse, error)
	Delete(ctx context.Context, id string, callerID string) error
	AssignRole(ctx context.Context, id string, req *dto.AssignRoleRequest, callerID string) error
	ResetPassword(ctx context.Context, id string, callerID string) (*dto.ResetPasswordResponse, error)
	ImportUsers(ctx context.Context, rows []ImportUserRow) (*dto.ImportUserResponse, error)
}

// ImportUserRow Excel 导入解析后的单行数据
type ImportUserRow struct {
	Row            int
	Name           string
	StudentID      string
	Email          string
	DepartmentName string
}

type userService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewUserService 创建 UserService 实例
func NewUserService(repo *repository.Repository, logger *zap.Logger) UserService {
	return &userService{repo: repo, logger: logger}
}

// ────────────────────── GetByID ──────────────────────

func (s *userService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	user, err := s.repo.User.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("查询用户失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toUserResponse(user), nil
}

// ────────────────────── List ──────────────────────

func (s *userService) List(ctx context.Context, req *dto.UserListRequest, callerRole, callerDeptID string) ([]dto.UserResponse, int64, error) {
	filters := &repository.UserListFilters{
		DepartmentID: req.DepartmentID,
		Role:         req.Role,
		Keyword:      req.Keyword,
	}

	// leader 自动过滤为本部门
	if callerRole == "leader" {
		filters.DepartmentID = callerDeptID
	}

	users, total, err := s.repo.User.ListWithFilters(ctx, filters, req.GetOffset(), req.GetPageSize())
	if err != nil {
		s.logger.Error("列出用户失败", zap.Error(err))
		return nil, 0, err
	}

	result := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		result = append(result, *s.toUserResponse(&u))
	}

	return result, total, nil
}

// ────────────────────── Update ──────────────────────

func (s *userService) Update(ctx context.Context, id string, req *dto.UpdateUserRequest, callerID, callerRole string) (*dto.UserResponse, error) {
	user, err := s.repo.User.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("查询用户失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	// 非管理员只能修改自己，且不能修改 department_id
	if callerRole != "admin" {
		if callerID != id {
			return nil, ErrNoPermission
		}
		if req.DepartmentID != nil {
			return nil, ErrNoPermission
		}
	}

	// 应用更新字段（仅更新非 nil 字段）
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		// 检查邮箱唯一性
		existing, err := s.repo.User.GetByEmail(ctx, *req.Email)
		if err == nil && existing.UserID != id {
			return nil, ErrEmailExists
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		user.Email = *req.Email
	}
	if req.DepartmentID != nil {
		// 验证部门存在
		if _, err := s.repo.Department.GetByID(ctx, *req.DepartmentID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrDepartmentNotFound
			}
			return nil, err
		}
		user.DepartmentID = *req.DepartmentID
	}

	user.UpdatedBy = &callerID

	if err := s.repo.User.Update(ctx, user); err != nil {
		s.logger.Error("更新用户失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	// 重新加载关联
	updated, err := s.repo.User.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toUserResponse(updated), nil
}

// ────────────────────── Delete ──────────────────────

func (s *userService) Delete(ctx context.Context, id string, callerID string) error {
	if id == callerID {
		return ErrUserSelfDelete
	}

	// 检查用户存在
	if _, err := s.repo.User.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		s.logger.Error("查询用户失败", zap.String("id", id), zap.Error(err))
		return err
	}

	if err := s.repo.User.Delete(ctx, id, callerID); err != nil {
		s.logger.Error("删除用户失败", zap.String("id", id), zap.Error(err))
		return err
	}

	return nil
}

// ────────────────────── AssignRole ──────────────────────

func (s *userService) AssignRole(ctx context.Context, id string, req *dto.AssignRoleRequest, callerID string) error {
	if id == callerID {
		return ErrUserSelfRoleChange
	}

	user, err := s.repo.User.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		s.logger.Error("查询用户失败", zap.String("id", id), zap.Error(err))
		return err
	}

	user.Role = req.Role
	user.UpdatedBy = &callerID

	if err := s.repo.User.Update(ctx, user); err != nil {
		s.logger.Error("分配角色失败", zap.String("id", id), zap.Error(err))
		return err
	}

	return nil
}

// ────────────────────── ResetPassword ──────────────────────

func (s *userService) ResetPassword(ctx context.Context, id string, callerID string) (*dto.ResetPasswordResponse, error) {
	user, err := s.repo.User.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.Error("查询用户失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	// 生成 8 位随机密码（保证包含字母和数字）
	tempPassword, err := generateTempPassword(8)
	if err != nil {
		s.logger.Error("生成临时密码失败", zap.Error(err))
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("密码哈希失败", zap.Error(err))
		return nil, err
	}

	user.PasswordHash = string(hash)
	user.MustChangePassword = true
	user.UpdatedBy = &callerID

	if err := s.repo.User.Update(ctx, user); err != nil {
		s.logger.Error("重置密码失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return &dto.ResetPasswordResponse{TempPassword: tempPassword}, nil
}

// ────────────────────── ImportUsers ──────────────────────

func (s *userService) ImportUsers(ctx context.Context, rows []ImportUserRow) (*dto.ImportUserResponse, error) {
	resp := &dto.ImportUserResponse{Total: len(rows)}

	// 预加载所有部门，便于按名称查找
	deptMap, err := s.buildDepartmentMap(ctx)
	if err != nil {
		s.logger.Error("加载部门列表失败", zap.Error(err))
		return nil, err
	}

	for _, row := range rows {
		// 校验必填字段
		if row.Name == "" || row.StudentID == "" || row.Email == "" || row.DepartmentName == "" {
			resp.Failed++
			resp.Errors = append(resp.Errors, dto.ImportUserError{
				Row: row.Row, Reason: "必填字段为空",
			})
			continue
		}

		// 查找部门
		dept, ok := deptMap[row.DepartmentName]
		if !ok {
			resp.Failed++
			resp.Errors = append(resp.Errors, dto.ImportUserError{
				Row: row.Row, Reason: fmt.Sprintf("部门不存在: %s", row.DepartmentName),
			})
			continue
		}

		// 检查学号唯一性
		if _, err := s.repo.User.GetByStudentID(ctx, row.StudentID); err == nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, dto.ImportUserError{
				Row: row.Row, Reason: fmt.Sprintf("学号已存在: %s", row.StudentID),
			})
			continue
		}

		// 检查邮箱唯一性
		if _, err := s.repo.User.GetByEmail(ctx, row.Email); err == nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, dto.ImportUserError{
				Row: row.Row, Reason: fmt.Sprintf("邮箱已存在: %s", row.Email),
			})
			continue
		}

		// 默认密码 = "Ec" + 学号后6位（保证满足8位最低长度 + 字母数字混合）
		defaultPwd := row.StudentID
		if len(defaultPwd) > 6 {
			defaultPwd = defaultPwd[len(defaultPwd)-6:]
		}
		defaultPwd = "Ec" + defaultPwd

		hash, err := bcrypt.GenerateFromPassword([]byte(defaultPwd), bcrypt.DefaultCost)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, dto.ImportUserError{
				Row: row.Row, Reason: "密码哈希失败",
			})
			continue
		}

		user := &model.User{
			Name:               row.Name,
			StudentID:          row.StudentID,
			Email:              row.Email,
			PasswordHash:       string(hash),
			Role:               "member",
			DepartmentID:       dept.DepartmentID,
			MustChangePassword: true,
		}

		if err := s.repo.User.Create(ctx, user); err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, dto.ImportUserError{
				Row: row.Row, Reason: fmt.Sprintf("数据库写入失败: %v", err),
			})
		} else {
			resp.Success++
		}
	}

	return resp, nil
}

// ── 内部辅助方法 ──

// toUserResponse 将 model.User 转换为 dto.UserResponse
func (s *userService) toUserResponse(user *model.User) *dto.UserResponse {
	var dept *dto.DepartmentResponse
	if user.Department != nil {
		dept = &dto.DepartmentResponse{
			ID:   user.Department.DepartmentID,
			Name: user.Department.Name,
		}
	}
	return &dto.UserResponse{
		ID:         user.UserID,
		Name:       user.Name,
		Email:      user.Email,
		StudentID:  user.StudentID,
		Role:       user.Role,
		Department: dept,
	}
}

// buildDepartmentMap 构建部门名称 -> 部门实体映射
func (s *userService) buildDepartmentMap(ctx context.Context) (map[string]*model.Department, error) {
	departments, err := s.repo.Department.List(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*model.Department, len(departments))
	for i := range departments {
		m[departments[i].Name] = &departments[i]
	}
	return m, nil
}

// generateTempPassword 生成指定长度的临时密码（保证包含字母和数字）
func generateTempPassword(length int) (string, error) {
	const letters = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
	const digits = "23456789"
	const all = letters + digits

	if length < 4 {
		length = 8
	}

	result := make([]byte, length)

	// 保证至少1个字母+1个数字
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
	if err != nil {
		return "", err
	}
	result[0] = letters[n.Int64()]

	n, err = rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
	if err != nil {
		return "", err
	}
	result[1] = digits[n.Int64()]

	// 剩余位随机填充
	for i := 2; i < length; i++ {
		n, err = rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		if err != nil {
			return "", err
		}
		result[i] = all[n.Int64()]
	}

	// Fisher-Yates 洗牌
	for i := length - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		result[i], result[j.Int64()] = result[j.Int64()], result[i]
	}

	return string(result), nil
}
