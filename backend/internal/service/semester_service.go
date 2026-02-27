package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 学期模块业务错误 ──

var (
	ErrSemesterNotFound    = errors.New("学期不存在")
	ErrSemesterDateInvalid = errors.New("学期结束日期必须晚于开始日期")
	ErrSemesterDateOverlap = errors.New("学期日期与已有学期重叠")
	ErrPhaseAdvanceInvalid = errors.New("阶段推进无效：前置条件未满足")
	ErrPhaseTransInvalid   = errors.New("无效的阶段跳转")
)

// SemesterService 学期业务接口
type SemesterService interface {
	Create(ctx context.Context, req *dto.CreateSemesterRequest, callerID string) (*dto.SemesterResponse, error)
	GetByID(ctx context.Context, id string) (*dto.SemesterResponse, error)
	GetCurrent(ctx context.Context) (*dto.SemesterResponse, error)
	List(ctx context.Context) ([]dto.SemesterResponse, error)
	Update(ctx context.Context, id string, req *dto.UpdateSemesterRequest, callerID string) (*dto.SemesterResponse, error)
	Activate(ctx context.Context, id string, callerID string) error
	Delete(ctx context.Context, id string, callerID string) error
	// ── 阶段推进 API ──
	CheckPhase(ctx context.Context, id string) (*dto.PhaseCheckResponse, error)
	AdvancePhase(ctx context.Context, id string, req *dto.AdvancePhaseRequest, callerID string) error
	// ── 值班人员管理 ──
	GetDutyMembers(ctx context.Context, semesterID string) ([]dto.DutyMemberItem, error)
	SetDutyMembers(ctx context.Context, semesterID string, req *dto.DutyMembersRequest, callerID string) error
	// ── 待办通知 ──
	GetPendingTodos(ctx context.Context, userID string) ([]dto.PendingTodoItem, error)
}

type semesterService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewSemesterService 创建 SemesterService 实例
func NewSemesterService(repo *repository.Repository, logger *zap.Logger) SemesterService {
	return &semesterService{repo: repo, logger: logger}
}

// ────────────────────── Create ──────────────────────

func (s *semesterService) Create(ctx context.Context, req *dto.CreateSemesterRequest, callerID string) (*dto.SemesterResponse, error) {
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, ErrSemesterDateInvalid
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, ErrSemesterDateInvalid
	}
	if !endDate.After(startDate) {
		return nil, ErrSemesterDateInvalid
	}

	// 检查日期是否与已有学期重叠
	overlap, err := s.repo.Semester.HasOverlap(ctx, req.StartDate, req.EndDate, "")
	if err != nil {
		s.logger.Error("检查学期重叠失败", zap.Error(err))
		return nil, err
	}
	if overlap {
		return nil, ErrSemesterDateOverlap
	}

	semester := &model.Semester{
		Name:          req.Name,
		StartDate:     startDate,
		EndDate:       endDate,
		FirstWeekType: req.FirstWeekType,
		IsActive:      false,
		Status:        "active",
		Phase:         model.SemesterPhaseConfiguring,
	}
	semester.CreatedBy = &callerID
	semester.UpdatedBy = &callerID

	if err := s.repo.Semester.Create(ctx, semester); err != nil {
		s.logger.Error("创建学期失败", zap.Error(err))
		return nil, err
	}

	return s.toSemesterResponse(semester), nil
}

// ────────────────────── GetByID ──────────────────────

func (s *semesterService) GetByID(ctx context.Context, id string) (*dto.SemesterResponse, error) {
	semester, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toSemesterResponse(semester), nil
}

// ────────────────────── GetCurrent ──────────────────────

func (s *semesterService) GetCurrent(ctx context.Context) (*dto.SemesterResponse, error) {
	semester, err := s.repo.Semester.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		s.logger.Error("查询当前学期失败", zap.Error(err))
		return nil, err
	}

	return s.toSemesterResponse(semester), nil
}

// ────────────────────── List ──────────────────────

func (s *semesterService) List(ctx context.Context) ([]dto.SemesterResponse, error) {
	semesters, err := s.repo.Semester.List(ctx)
	if err != nil {
		s.logger.Error("列出学期失败", zap.Error(err))
		return nil, err
	}

	result := make([]dto.SemesterResponse, 0, len(semesters))
	for i := range semesters {
		result = append(result, *s.toSemesterResponse(&semesters[i]))
	}

	return result, nil
}

// ────────────────────── Update ──────────────────────

func (s *semesterService) Update(ctx context.Context, id string, req *dto.UpdateSemesterRequest, callerID string) (*dto.SemesterResponse, error) {
	semester, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	if req.Name != nil {
		semester.Name = *req.Name
	}
	if req.StartDate != nil {
		startDate, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			return nil, ErrSemesterDateInvalid
		}
		semester.StartDate = startDate
	}
	if req.EndDate != nil {
		endDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, ErrSemesterDateInvalid
		}
		semester.EndDate = endDate
	}
	if !semester.EndDate.After(semester.StartDate) {
		return nil, ErrSemesterDateInvalid
	}

	// 检查日期是否与已有学期重叠（排除自身）
	overlap, err := s.repo.Semester.HasOverlap(ctx,
		semester.StartDate.Format("2006-01-02"),
		semester.EndDate.Format("2006-01-02"), id)
	if err != nil {
		s.logger.Error("检查学期重叠失败", zap.Error(err))
		return nil, err
	}
	if overlap {
		return nil, ErrSemesterDateOverlap
	}

	if req.FirstWeekType != nil {
		semester.FirstWeekType = *req.FirstWeekType
	}
	if req.Status != nil {
		semester.Status = *req.Status
	}

	semester.UpdatedBy = &callerID

	if err := s.repo.Semester.Update(ctx, semester); err != nil {
		s.logger.Error("更新学期失败", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return s.toSemesterResponse(semester), nil
}

// ────────────────────── Activate ──────────────────────

func (s *semesterService) Activate(ctx context.Context, id string, callerID string) error {
	semester, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.String("id", id), zap.Error(err))
		return err
	}

	// 使用事务保证 ClearActive + Update 的原子性
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error("开启事务失败", zap.Error(err))
		return err
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

	// 先将所有学期置为非活动
	if err := txRepo.Semester.ClearActive(ctx); err != nil {
		if tx != nil {
			tx.Rollback()
		}
		s.logger.Error("清除活动学期失败", zap.Error(err))
		return err
	}

	// 设置目标学期为活动，默认阶段为 configuring
	semester.IsActive = true
	semester.Phase = model.SemesterPhaseConfiguring
	semester.UpdatedBy = &callerID

	if err := txRepo.Semester.Update(ctx, semester); err != nil {
		if tx != nil {
			tx.Rollback()
		}
		s.logger.Error("激活学期失败", zap.String("id", id), zap.Error(err))
		return err
	}

	if tx != nil {
		if err := tx.Commit().Error; err != nil {
			s.logger.Error("提交事务失败", zap.Error(err))
			return err
		}
	}

	return nil
}

// ────────────────────── Delete ──────────────────────

func (s *semesterService) Delete(ctx context.Context, id string, callerID string) error {
	_, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.String("id", id), zap.Error(err))
		return err
	}

	if err := s.repo.Semester.Delete(ctx, id, callerID); err != nil {
		s.logger.Error("删除学期失败", zap.String("id", id), zap.Error(err))
		return err
	}

	return nil
}

// ── 内部辅助方法 ──

func (s *semesterService) toSemesterResponse(semester *model.Semester) *dto.SemesterResponse {
	return &dto.SemesterResponse{
		ID:            semester.SemesterID,
		Name:          semester.Name,
		StartDate:     semester.StartDate.Format("2006-01-02"),
		EndDate:       semester.EndDate.Format("2006-01-02"),
		FirstWeekType: semester.FirstWeekType,
		IsActive:      semester.IsActive,
		Status:        semester.Status,
		Phase:         semester.Phase,
		CreatedAt:     semester.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     semester.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ────────────────────── CheckPhase ──────────────────────

// phaseOrder 定义阶段顺序
var phaseOrder = []string{
	model.SemesterPhaseConfiguring,
	model.SemesterPhaseCollecting,
	model.SemesterPhaseScheduling,
	model.SemesterPhasePublished,
}

func phaseIndex(phase string) int {
	for i, p := range phaseOrder {
		if p == phase {
			return i
		}
	}
	return -1
}

func (s *semesterService) CheckPhase(ctx context.Context, id string) (*dto.PhaseCheckResponse, error) {
	semester, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		return nil, err
	}

	resp := &dto.PhaseCheckResponse{
		CurrentPhase: semester.Phase,
		CanAdvance:   true,
		Checks:       []dto.PhaseCheckItem{},
	}

	switch semester.Phase {
	case model.SemesterPhaseConfiguring:
		// 检查：至少1个时间段、至少1个地点、至少1名值班人员
		timeSlots, _ := s.repo.TimeSlot.List(ctx, id, nil)
		locations, _ := s.repo.Location.List(ctx, false)
		dutyCount, _ := s.repo.UserSemesterAssignment.CountDutyRequired(ctx, id)

		tsCheck := dto.PhaseCheckItem{Label: "时间段配置", Passed: len(timeSlots) > 0}
		if !tsCheck.Passed {
			tsCheck.Message = "至少需要配置1个时间段"
			resp.CanAdvance = false
		}
		resp.Checks = append(resp.Checks, tsCheck)

		locCheck := dto.PhaseCheckItem{Label: "地点配置", Passed: len(locations) > 0}
		if !locCheck.Passed {
			locCheck.Message = "至少需要配置1个地点"
			resp.CanAdvance = false
		}
		resp.Checks = append(resp.Checks, locCheck)

		dutyCheck := dto.PhaseCheckItem{Label: "值班人员", Passed: dutyCount > 0}
		if !dutyCheck.Passed {
			dutyCheck.Message = "至少需要选定1名值班人员"
			resp.CanAdvance = false
		}
		resp.Checks = append(resp.Checks, dutyCheck)

	case model.SemesterPhaseCollecting:
		// 检查：所有值班人员已提交时间表
		total, _ := s.repo.UserSemesterAssignment.CountDutyRequired(ctx, id)
		submitted, _ := s.repo.UserSemesterAssignment.CountDutyRequiredSubmitted(ctx, id)

		check := dto.PhaseCheckItem{
			Label:  "时间表提交",
			Passed: total > 0 && submitted == total,
		}
		if !check.Passed {
			check.Message = fmt.Sprintf("已提交 %d / %d 人", submitted, total)
			resp.CanAdvance = false
		}
		resp.Checks = append(resp.Checks, check)

	case model.SemesterPhaseScheduling:
		// 检查：存在排班表
		schedule, _ := s.repo.Schedule.GetBySemester(ctx, id)
		check := dto.PhaseCheckItem{
			Label:  "排班表",
			Passed: schedule != nil,
		}
		if !check.Passed {
			check.Message = "尚未生成排班表"
			resp.CanAdvance = false
		}
		resp.Checks = append(resp.Checks, check)

	case model.SemesterPhasePublished:
		// 已发布，无需推进
		resp.CanAdvance = false
	}

	return resp, nil
}

// ────────────────────── AdvancePhase ──────────────────────

func (s *semesterService) AdvancePhase(ctx context.Context, id string, req *dto.AdvancePhaseRequest, callerID string) error {
	semester, err := s.repo.Semester.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSemesterNotFound
		}
		return err
	}

	currentIdx := phaseIndex(semester.Phase)
	targetIdx := phaseIndex(req.TargetPhase)

	if currentIdx < 0 || targetIdx < 0 {
		return ErrPhaseTransInvalid
	}

	// 允许回退（任意阶段可回退到前序阶段，保留已有数据）
	if targetIdx < currentIdx {
		semester.Phase = req.TargetPhase
		semester.UpdatedBy = &callerID
		return s.repo.Semester.Update(ctx, semester)
	}

	// 前进只允许+1步
	if targetIdx != currentIdx+1 {
		return ErrPhaseTransInvalid
	}

	// 前进需检查条件
	checkResp, err := s.CheckPhase(ctx, id)
	if err != nil {
		return err
	}
	if !checkResp.CanAdvance {
		return ErrPhaseAdvanceInvalid
	}

	semester.Phase = req.TargetPhase
	semester.UpdatedBy = &callerID
	return s.repo.Semester.Update(ctx, semester)
}

// ────────────────────── GetDutyMembers ──────────────────────

func (s *semesterService) GetDutyMembers(ctx context.Context, semesterID string) ([]dto.DutyMemberItem, error) {
	_, err := s.repo.Semester.GetByID(ctx, semesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		return nil, err
	}

	assignments, err := s.repo.UserSemesterAssignment.ListBySemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	items := make([]dto.DutyMemberItem, 0, len(assignments))
	for _, a := range assignments {
		item := dto.DutyMemberItem{
			UserID:       a.UserID,
			DutyRequired: a.DutyRequired,
		}
		if a.User != nil {
			item.Name = a.User.Name
			item.StudentID = a.User.StudentID
			item.DepartmentID = a.User.DepartmentID
			if a.User.Department != nil {
				item.DepartmentName = a.User.Department.Name
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// ────────────────────── SetDutyMembers ──────────────────────

func (s *semesterService) SetDutyMembers(ctx context.Context, semesterID string, req *dto.DutyMembersRequest, callerID string) error {
	_, err := s.repo.Semester.GetByID(ctx, semesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSemesterNotFound
		}
		return err
	}

	// 先将该学期所有用户的 duty_required 设为 false
	allAssignments, err := s.repo.UserSemesterAssignment.ListBySemester(ctx, semesterID)
	if err != nil {
		return err
	}
	for _, a := range allAssignments {
		if a.DutyRequired {
			if err := s.repo.UserSemesterAssignment.UpdateDutyRequired(ctx, a.AssignmentID, false, callerID); err != nil {
				return err
			}
		}
	}

	// Upsert 选定的值班人员
	if len(req.UserIDs) > 0 {
		if err := s.repo.UserSemesterAssignment.BatchUpsert(ctx, semesterID, req.UserIDs, true, callerID); err != nil {
			return err
		}
	}
	return nil
}

// ────────────────────── GetPendingTodos ──────────────────────

func (s *semesterService) GetPendingTodos(ctx context.Context, userID string) ([]dto.PendingTodoItem, error) {
	semester, err := s.repo.Semester.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []dto.PendingTodoItem{}, nil // 无活跃学期，无待办
		}
		return nil, err
	}

	var todos []dto.PendingTodoItem

	switch semester.Phase {
	case model.SemesterPhaseCollecting:
		// 检查用户是否需要值班
		assignment, err := s.repo.UserSemesterAssignment.GetByUserAndSemester(ctx, userID, semester.SemesterID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return todos, nil // 不在值班池中
			}
			return nil, err
		}
		if assignment.DutyRequired {
			if assignment.TimetableStatus == model.TimetableStatusNotSubmitted {
				todos = append(todos, dto.PendingTodoItem{
					Type:    "submit_timetable",
					Title:   "提交时间表",
					Message: "请导入课表并标记不可用时间后提交",
				})
			} else {
				todos = append(todos, dto.PendingTodoItem{
					Type:    "timetable_submitted",
					Title:   "时间表已提交",
					Message: "已提交，正在等待所有成员完成提交",
				})
			}
		}

	case model.SemesterPhaseScheduling:
		todos = append(todos, dto.PendingTodoItem{
			Type:    "waiting_schedule",
			Title:   "排班进行中",
			Message: "管理员正在安排排班，请耐心等待",
		})

	case model.SemesterPhasePublished:
		todos = append(todos, dto.PendingTodoItem{
			Type:    "schedule_published",
			Title:   "排班已发布",
			Message: "本学期排班表已发布，请查看你的值班安排",
		})
	}

	return todos, nil
}
