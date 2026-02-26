package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 部门模块业务错误 ──

var (
	ErrDepartmentNameExists      = errors.New("部门名称已存在")
	ErrDepartmentHasMembers      = errors.New("部门下存在成员，无法删除")
	ErrDepartmentInactive        = errors.New("部门已停用")
	ErrDutyMemberNotInDepartment = errors.New("指定用户不属于该部门")
)

// DepartmentService 部门业务接口
type DepartmentService interface {
	Create(ctx context.Context, req *dto.CreateDepartmentRequest, callerID string) (*dto.DepartmentDetailResponse, error)
	GetByID(ctx context.Context, id string) (*dto.DepartmentDetailResponse, error)
	List(ctx context.Context, req *dto.DepartmentListRequest) ([]dto.DepartmentDetailResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateDepartmentRequest, callerID string) (*dto.DepartmentDetailResponse, error)
	Delete(ctx context.Context, id string, callerID string) error
	// GetMembers 获取部门成员列表（含学期分配状态）
	GetMembers(ctx context.Context, departmentID, semesterID string) ([]dto.DepartmentMemberResponse, error)
	// SetDutyMembers 批量设置部门值班人员
	SetDutyMembers(ctx context.Context, departmentID string, req *dto.SetDutyMembersRequest, callerID string) (*dto.SetDutyMembersResponse, error)
}

type departmentService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewDepartmentService 创建 DepartmentService 实例
func NewDepartmentService(repo *repository.Repository, logger *zap.Logger) DepartmentService {
	return &departmentService{repo: repo, logger: logger}
}

// ────────────────────── Create ──────────────────────

func (s *departmentService) Create(ctx context.Context, req *dto.CreateDepartmentRequest, callerID string) (*dto.DepartmentDetailResponse, error) {
	// 检查名称唯一性
	existing, err := s.repo.Department.GetByName(ctx, req.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("查询部门失败", zap.Error(err))
		return nil, err
	}
	if existing != nil {
		return nil, ErrDepartmentNameExists
	}

	dept := &model.Department{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
	}
	dept.CreatedBy = &callerID
	dept.UpdatedBy = &callerID

	if err := s.repo.Department.Create(ctx, dept); err != nil {
		s.logger.Error("创建部门失败", zap.Error(err))
		return nil, err
	}

	return s.toDepartmentDetailResponse(ctx, dept), nil
}

// ────────────────────── GetByID ──────────────────────

func (s *departmentService) GetByID(ctx context.Context, id string) (*dto.DepartmentDetailResponse, error) {
	dept, err := s.repo.Department.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDepartmentNotFound
		}
		s.logger.Error("查询部门失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toDepartmentDetailResponse(ctx, dept), nil
}

// ────────────────────── List ──────────────────────

func (s *departmentService) List(ctx context.Context, req *dto.DepartmentListRequest) ([]dto.DepartmentDetailResponse, error) {
	var depts []model.Department
	var err error

	if req.IncludeInactive {
		depts, err = s.repo.Department.ListAll(ctx)
	} else {
		depts, err = s.repo.Department.List(ctx)
	}
	if err != nil {
		s.logger.Error("列出部门失败", zap.Error(err))
		return nil, err
	}

	// 批量查询成员数，避免 N+1 查询问题
	deptIDs := make([]string, 0, len(depts))
	for _, d := range depts {
		deptIDs = append(deptIDs, d.DepartmentID)
	}
	countMap, err := s.repo.Department.BatchCountMembers(ctx, deptIDs)
	if err != nil {
		s.logger.Warn("批量查询成员数失败，回退为0", zap.Error(err))
		countMap = make(map[string]int64)
	}

	result := make([]dto.DepartmentDetailResponse, 0, len(depts))
	for i := range depts {
		result = append(result, dto.DepartmentDetailResponse{
			ID:          depts[i].DepartmentID,
			Name:        depts[i].Name,
			Description: depts[i].Description,
			IsActive:    depts[i].IsActive,
			MemberCount: countMap[depts[i].DepartmentID],
			CreatedAt:   depts[i].CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   depts[i].UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return result, nil
}

// ────────────────────── Update ──────────────────────

func (s *departmentService) Update(ctx context.Context, id string, req *dto.UpdateDepartmentRequest, callerID string) (*dto.DepartmentDetailResponse, error) {
	dept, err := s.repo.Department.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDepartmentNotFound
		}
		s.logger.Error("查询部门失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	// 如果更新名称，检查唯一性
	if req.Name != nil && *req.Name != dept.Name {
		existing, err := s.repo.Department.GetByName(ctx, *req.Name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if existing != nil {
			return nil, ErrDepartmentNameExists
		}
		dept.Name = *req.Name
	}

	if req.Description != nil {
		dept.Description = *req.Description
	}
	if req.IsActive != nil {
		dept.IsActive = *req.IsActive
	}

	dept.UpdatedBy = &callerID

	if err := s.repo.Department.Update(ctx, dept); err != nil {
		s.logger.Error("更新部门失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toDepartmentDetailResponse(ctx, dept), nil
}

// ────────────────────── Delete ──────────────────────

func (s *departmentService) Delete(ctx context.Context, id string, callerID string) error {
	dept, err := s.repo.Department.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDepartmentNotFound
		}
		s.logger.Error("查询部门失败", zap.String("id", id), zap.Error(err))
		return err
	}

	// 检查部门下是否有成员
	count, err := s.repo.Department.CountMembers(ctx, dept.DepartmentID)
	if err != nil {
		s.logger.Error("查询部门成员数失败", zap.String("id", id), zap.Error(err))
		return err
	}
	if count > 0 {
		return ErrDepartmentHasMembers
	}

	if err := s.repo.Department.Delete(ctx, id, callerID); err != nil {
		s.logger.Error("删除部门失败", zap.String("id", id), zap.Error(err))
		return err
	}

	return nil
}

// ── 内部辅助方法 ──

func (s *departmentService) toDepartmentDetailResponse(ctx context.Context, dept *model.Department) *dto.DepartmentDetailResponse {
	memberCount, _ := s.repo.Department.CountMembers(ctx, dept.DepartmentID)
	return &dto.DepartmentDetailResponse{
		ID:          dept.DepartmentID,
		Name:        dept.Name,
		Description: dept.Description,
		IsActive:    dept.IsActive,
		MemberCount: memberCount,
		CreatedAt:   dept.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   dept.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ═══════════════════════════════════════════════════════════
// GetMembers — 获取部门成员列表（含学期分配状态）
// ═══════════════════════════════════════════════════════════
//
// 设计说明：
//   - 如果传了 semesterID，会查询 user_semester_assignments 表获取值班标记和提交状态
//   - 如果未传 semesterID，duty_required 和 timetable_status 默认为 false / "not_submitted"
//   - 这样前端在"勾选值班人员"页面可同时展示成员和当前学期的值班状态

func (s *departmentService) GetMembers(ctx context.Context, departmentID, semesterID string) ([]dto.DepartmentMemberResponse, error) {
	// 校验部门存在
	if _, err := s.repo.Department.GetByID(ctx, departmentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}

	// 获取部门下所有用户
	filters := &repository.UserListFilters{DepartmentID: departmentID}
	users, _, err := s.repo.User.ListWithFilters(ctx, filters, 0, 1000)
	if err != nil {
		s.logger.Error("查询部门成员失败", zap.Error(err))
		return nil, err
	}

	// 如果有学期 ID，构建 assignment map
	assignmentMap := make(map[string]*model.UserSemesterAssignment)
	if semesterID != "" {
		assignments, err := s.repo.UserSemesterAssignment.ListByDepartmentAndSemester(ctx, departmentID, semesterID)
		if err != nil {
			s.logger.Warn("查询学期分配记录失败", zap.Error(err))
		} else {
			for i := range assignments {
				assignmentMap[assignments[i].UserID] = &assignments[i]
			}
		}
	}

	result := make([]dto.DepartmentMemberResponse, 0, len(users))
	for _, u := range users {
		member := dto.DepartmentMemberResponse{
			UserID:          u.UserID,
			Name:            u.Name,
			StudentID:       u.StudentID,
			Email:           u.Email,
			Role:            u.Role,
			DutyRequired:    false,
			TimetableStatus: "not_submitted",
		}
		if a, ok := assignmentMap[u.UserID]; ok {
			member.DutyRequired = a.DutyRequired
			member.TimetableStatus = a.TimetableStatus
		}
		result = append(result, member)
	}

	return result, nil
}

// ═══════════════════════════════════════════════════════════
// SetDutyMembers — 批量设置部门值班人员
// ═══════════════════════════════════════════════════════════
//
// 设计说明：
//   - 全量替换策略：传入的 user_ids 即为该部门本学期的值班成员
//   - 先将该部门下所有已有分配记录的 duty_required 置 false
//   - 再对传入的 user_ids 执行 upsert（存在则更新，不存在则创建）
//   - 确保传入的用户确实属于该部门（安全校验）

func (s *departmentService) SetDutyMembers(ctx context.Context, departmentID string, req *dto.SetDutyMembersRequest, callerID string) (*dto.SetDutyMembersResponse, error) {
	// 1. 校验部门存在
	dept, err := s.repo.Department.GetByID(ctx, departmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}

	// 2. 校验学期存在
	if _, err := s.repo.Semester.GetByID(ctx, req.SemesterID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		return nil, err
	}

	// 3. 批量校验所有 user_ids 属于该部门
	users, err := s.repo.User.ListByIDs(ctx, req.UserIDs)
	if err != nil {
		s.logger.Error("批量查询用户失败", zap.Error(err))
		return nil, err
	}
	userMap := make(map[string]*model.User, len(users))
	for i := range users {
		userMap[users[i].UserID] = &users[i]
	}
	for _, uid := range req.UserIDs {
		user, ok := userMap[uid]
		if !ok {
			return nil, ErrUserNotFound
		}
		if user.DepartmentID != departmentID {
			return nil, ErrDutyMemberNotInDepartment
		}
	}

	// 4. 先清除该部门下当前学期所有 duty_required
	existingAssignments, err := s.repo.UserSemesterAssignment.ListByDepartmentAndSemester(ctx, departmentID, req.SemesterID)
	if err != nil {
		s.logger.Error("查询现有分配失败", zap.Error(err))
		return nil, err
	}
	for _, a := range existingAssignments {
		if a.DutyRequired {
			if err := s.repo.UserSemesterAssignment.UpdateDutyRequired(ctx, a.AssignmentID, false, callerID); err != nil {
				s.logger.Error("清除值班标记失败", zap.Error(err))
				return nil, err
			}
		}
	}

	// 5. 批量 upsert 传入的 user_ids
	if err := s.repo.UserSemesterAssignment.BatchUpsert(ctx, req.SemesterID, req.UserIDs, true, callerID); err != nil {
		s.logger.Error("设置值班人员失败", zap.Error(err))
		return nil, err
	}

	return &dto.SetDutyMembersResponse{
		DepartmentID:   dept.DepartmentID,
		DepartmentName: dept.Name,
		SemesterID:     req.SemesterID,
		TotalSet:       len(req.UserIDs),
	}, nil
}
