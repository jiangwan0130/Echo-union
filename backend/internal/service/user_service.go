package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/xuri/excelize/v2"
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
	CreateUser(ctx context.Context, req *dto.CreateUserRequest, callerID string) (*dto.CreateUserResponse, error)
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
	List(ctx context.Context, req *dto.UserListRequest, callerRole, callerDeptID string) ([]dto.UserResponse, int64, error)
	Update(ctx context.Context, id string, req *dto.UpdateUserRequest, callerID, callerRole string) (*dto.UserResponse, error)
	Delete(ctx context.Context, id string, callerID string) error
	AssignRole(ctx context.Context, id string, req *dto.AssignRoleRequest, callerID string) error
	ResetPassword(ctx context.Context, id string, callerID string) (*dto.ResetPasswordResponse, error)
	ParseImportFile(reader io.Reader) ([]ImportUserRow, error)
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

// ────────────────────── CreateUser ──────────────────────

func (s *userService) CreateUser(ctx context.Context, req *dto.CreateUserRequest, callerID string) (*dto.CreateUserResponse, error) {
	// 检查学号唯一性
	if _, err := s.repo.User.GetByStudentID(ctx, req.StudentID); err == nil {
		return nil, ErrStudentIDExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 检查邮箱唯一性
	if _, err := s.repo.User.GetByEmail(ctx, req.Email); err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 检查部门存在
	if _, err := s.repo.Department.GetByID(ctx, req.DepartmentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}

	// 默认密码 = "Ec" + 学号后6位（与批量导入逻辑一致）
	defaultPwd := req.StudentID
	if len(defaultPwd) > 6 {
		defaultPwd = defaultPwd[len(defaultPwd)-6:]
	}
	defaultPwd = "Ec" + defaultPwd

	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPwd), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("密码哈希失败", zap.Error(err))
		return nil, err
	}

	user := &model.User{
		Name:               req.Name,
		StudentID:          req.StudentID,
		Email:              req.Email,
		PasswordHash:       string(hash),
		Role:               req.Role,
		DepartmentID:       req.DepartmentID,
		MustChangePassword: true,
		VersionedModel:     model.VersionedModel{SoftDeleteModel: model.SoftDeleteModel{BaseModel: model.BaseModel{CreatedBy: &callerID}}},
	}

	if err := s.repo.User.Create(ctx, user); err != nil {
		s.logger.Error("创建用户失败", zap.Error(err))
		return nil, err
	}

	// 重新加载以获取关联数据（部门等）
	created, err := s.repo.User.GetByID(ctx, user.UserID)
	if err != nil {
		return nil, err
	}

	return &dto.CreateUserResponse{
		User:         s.toUserResponse(created),
		TempPassword: defaultPwd,
	}, nil
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

// ────────────────────── ParseImportFile ──────────────────────

const maxImportRows = 1000

var (
	ErrImportNoData      = errors.New("Excel文件无数据行（第一行为表头）")
	ErrImportTooManyRows = fmt.Errorf("数据行数超过上限 %d 行", maxImportRows)
	ErrImportBadHeader   = errors.New("Excel表头缺少必要列（姓名/学号/邮箱/部门）")
)

// ParseImportFile 解析导入 Excel 文件，返回解析后的行数据
func (s *userService) ParseImportFile(reader io.Reader) ([]ImportUserRow, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("无法解析Excel文件: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	excelRows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败: %w", err)
	}

	if len(excelRows) < 2 {
		return nil, ErrImportNoData
	}

	// 解析表头（支持灵活列序）
	colIndex := parseHeaderIndex(excelRows[0])
	if colIndex["name"] < 0 || colIndex["student_id"] < 0 || colIndex["email"] < 0 || colIndex["department"] < 0 {
		return nil, ErrImportBadHeader
	}

	var rows []ImportUserRow
	for i := 1; i < len(excelRows); i++ {
		row := excelRows[i]
		item := ImportUserRow{Row: i + 1}

		if idx := colIndex["name"]; idx < len(row) {
			item.Name = strings.TrimSpace(row[idx])
		}
		if idx := colIndex["student_id"]; idx < len(row) {
			item.StudentID = strings.TrimSpace(row[idx])
		}
		if idx := colIndex["email"]; idx < len(row) {
			item.Email = strings.TrimSpace(row[idx])
		}
		if idx := colIndex["department"]; idx < len(row) {
			item.DepartmentName = strings.TrimSpace(row[idx])
		}

		// 跳过全空行
		if item.Name == "" && item.StudentID == "" && item.Email == "" && item.DepartmentName == "" {
			continue
		}

		rows = append(rows, item)
	}

	if len(rows) == 0 {
		return nil, ErrImportNoData
	}
	if len(rows) > maxImportRows {
		return nil, ErrImportTooManyRows
	}

	return rows, nil
}

// parseHeaderIndex 解析 Excel 表头，返回列名 -> 列索引映射
func parseHeaderIndex(header []string) map[string]int {
	idx := map[string]int{
		"name":       -1,
		"student_id": -1,
		"email":      -1,
		"department": -1,
	}
	for i, h := range header {
		lower := strings.ToLower(strings.TrimSpace(h))
		switch {
		case lower == "姓名" || lower == "name":
			idx["name"] = i
		case lower == "学号" || lower == "student_id":
			idx["student_id"] = i
		case lower == "邮箱" || lower == "email":
			idx["email"] = i
		case lower == "部门" || lower == "department":
			idx["department"] = i
		}
	}
	return idx
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

	// 第一阶段：数据预校验（不接触数据库写操作）
	type validatedRow struct {
		row  ImportUserRow
		dept *model.Department
		hash []byte
	}
	var validRows []validatedRow

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

		validRows = append(validRows, validatedRow{row: row, dept: dept, hash: hash})
	}

	// 第二阶段：在事务中批量创建所有通过校验的用户
	if len(validRows) > 0 {
		tx, err := s.repo.BeginTx(ctx)
		if err != nil {
			s.logger.Error("开启事务失败", zap.Error(err))
			return nil, err
		}
		defer func() {
			if r := recover(); r != nil {
				if tx != nil {
					tx.Rollback()
				}
				panic(r)
			}
		}()

		txRepo := s.repo.WithTx(tx)

		for _, vr := range validRows {
			user := &model.User{
				Name:               vr.row.Name,
				StudentID:          vr.row.StudentID,
				Email:              vr.row.Email,
				PasswordHash:       string(vr.hash),
				Role:               "member",
				DepartmentID:       vr.dept.DepartmentID,
				MustChangePassword: true,
			}

			if err := txRepo.User.Create(ctx, user); err != nil {
				// 事务中任一写入失败则全部回滚
				if tx != nil {
					tx.Rollback()
				}
				s.logger.Error("导入用户写入失败，事务回滚",
					zap.Int("row", vr.row.Row), zap.Error(err))
				return nil, fmt.Errorf("第 %d 行写入数据库失败，已回滚全部导入: %w", vr.row.Row, err)
			}
			resp.Success++
		}

		if tx != nil {
			if err := tx.Commit().Error; err != nil {
				s.logger.Error("提交事务失败", zap.Error(err))
				return nil, err
			}
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
