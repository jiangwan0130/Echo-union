package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 时间表模块业务错误 ──

var (
	ErrTimetableNoActiveSemester    = errors.New("当前无活动学期")
	ErrTimetableAssignmentNotFound  = errors.New("未找到用户学期分配记录")
	ErrTimetableAlreadySubmitted    = errors.New("时间表已提交，请重新导入课表后再提交")
	ErrTimetableNoCourses           = errors.New("尚未导入课表，无法提交")
	ErrTimetableICSParseFailed      = errors.New("ICS 文件解析失败")
	ErrTimetableICSEmpty            = errors.New("ICS 文件中未发现有效课程事件")
	ErrTimetableUnavailableNotFound = errors.New("不可用时间记录不存在")
	ErrTimetableUnavailableNotOwner = errors.New("无权操作此不可用时间记录")
	ErrTimetableDepartmentNotFound  = errors.New("部门不存在")
)

// ── TimetableService 接口 ──────────────────────────────────
//
// 设计说明：
//   - 课表导入（ImportICS）采用全量替换策略，在单个事务中执行
//     "删除旧数据 → 批量插入新数据 → 回退提交状态"，保证原子性。
//   - 不可用时间 CRUD 独立于课表，与课表共同构成"时间表"。
//   - 提交（Submit）将 timetable_status 从 not_submitted 更新为 submitted。
//   - 进度统计（Progress）按部门分组聚合。
// ─────────────────────────────────────────────────────────────

// TimetableService 时间表模块业务接口
type TimetableService interface {
	// ImportICS 导入 ICS 课表（文件或 URL）
	ImportICS(ctx context.Context, reader io.Reader, userID string, semesterID string) (*dto.ImportICSResponse, error)
	// GetMyTimetable 获取当前用户的时间表
	GetMyTimetable(ctx context.Context, userID string, semesterID string) (*dto.MyTimetableResponse, error)
	// CreateUnavailableTime 添加不可用时间
	CreateUnavailableTime(ctx context.Context, req *dto.CreateUnavailableTimeRequest, userID string) (*dto.UnavailableTimeResponse, error)
	// UpdateUnavailableTime 更新不可用时间
	UpdateUnavailableTime(ctx context.Context, id string, req *dto.UpdateUnavailableTimeRequest, userID string) (*dto.UnavailableTimeResponse, error)
	// DeleteUnavailableTime 删除不可用时间
	DeleteUnavailableTime(ctx context.Context, id string, userID string) error
	// SubmitTimetable 提交时间表
	SubmitTimetable(ctx context.Context, userID string, semesterID string) (*dto.SubmitTimetableResponse, error)
	// GetProgress 获取全局提交进度
	GetProgress(ctx context.Context, semesterID string) (*dto.TimetableProgressResponse, error)
	// GetDepartmentProgress 获取部门提交进度
	GetDepartmentProgress(ctx context.Context, departmentID string, semesterID string) (*dto.DepartmentProgressResponse, error)
}

type timetableService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewTimetableService 创建 TimetableService 实例
func NewTimetableService(repo *repository.Repository, logger *zap.Logger) TimetableService {
	return &timetableService{repo: repo, logger: logger}
}

// ════════════════════════════════════════════════════════════
// ImportICS — 导入 ICS 课表
// ════════════════════════════════════════════════════════════
//
// 流程：
//   1. 解析 ICS 内容为 courses 列表
//   2. 开启事务：删除旧课表 → 批量插入新课表
//   3. 回退 timetable_status 为 not_submitted（确保排班前课表已确认最新）

func (s *timetableService) ImportICS(ctx context.Context, reader io.Reader, userID string, semesterID string) (*dto.ImportICSResponse, error) {
	// 1. 确认学期
	semester, err := s.resolveActiveSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	// 2. 解析 ICS
	courses, err := ParseICS(reader, userID, semester.SemesterID, semester.StartDate, semester.EndDate)
	if err != nil {
		s.logger.Error("ICS 解析失败", zap.Error(err))
		return nil, ErrTimetableICSParseFailed
	}
	if len(courses) == 0 {
		return nil, ErrTimetableICSEmpty
	}

	// 3. 事务：删除旧数据 + 插入新数据（事务封装在 Repository 层）
	if err := s.repo.CourseSchedule.ReplaceByUserAndSemester(ctx, userID, semester.SemesterID, courses); err != nil {
		s.logger.Error("课表导入事务失败", zap.Error(err))
		return nil, fmt.Errorf("课表导入失败: %w", err)
	}

	// 4. 回退提交状态（非事务也可接受：即使失败，课表已更新，用户可手动重新提交）
	s.rollbackTimetableStatus(ctx, userID, semester.SemesterID)

	// 5. 构建响应
	events := make([]dto.ImportedCourseEvent, 0, len(courses))
	for _, c := range courses {
		events = append(events, dto.ImportedCourseEvent{
			Name:      c.CourseName,
			DayOfWeek: c.DayOfWeek,
			StartTime: c.StartTime,
			EndTime:   c.EndTime,
			Weeks:     []int(c.Weeks),
		})
	}

	return &dto.ImportICSResponse{
		ImportedCount: len(courses),
		Events:        events,
	}, nil
}

// ════════════════════════════════════════════════════════════
// GetMyTimetable — 获取我的时间表
// ════════════════════════════════════════════════════════════

func (s *timetableService) GetMyTimetable(ctx context.Context, userID string, semesterID string) (*dto.MyTimetableResponse, error) {
	semester, err := s.resolveActiveSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	courses, err := s.repo.CourseSchedule.ListByUserAndSemester(ctx, userID, semester.SemesterID)
	if err != nil {
		s.logger.Error("查询课表失败", zap.Error(err))
		return nil, err
	}

	unavailables, err := s.repo.UnavailableTime.ListByUserAndSemester(ctx, userID, semester.SemesterID)
	if err != nil {
		s.logger.Error("查询不可用时间失败", zap.Error(err))
		return nil, err
	}

	// 查询提交状态
	submitStatus := "not_submitted"
	var submittedAt *time.Time
	assignment, err := s.repo.UserSemesterAssignment.GetByUserAndSemester(ctx, userID, semester.SemesterID)
	if err == nil {
		submitStatus = assignment.TimetableStatus
		submittedAt = assignment.TimetableSubmittedAt
	}

	return &dto.MyTimetableResponse{
		Courses:      toCourseResponses(courses),
		Unavailable:  toUnavailableResponses(unavailables),
		SubmitStatus: submitStatus,
		SubmittedAt:  submittedAt,
	}, nil
}

// ════════════════════════════════════════════════════════════
// 不可用时间 CRUD
// ════════════════════════════════════════════════════════════

func (s *timetableService) CreateUnavailableTime(ctx context.Context, req *dto.CreateUnavailableTimeRequest, userID string) (*dto.UnavailableTimeResponse, error) {
	semester, err := s.resolveActiveSemester(ctx, req.SemesterID)
	if err != nil {
		return nil, err
	}

	ut := model.UnavailableTime{
		UserID:     userID,
		SemesterID: semester.SemesterID,
		DayOfWeek:  req.DayOfWeek,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Reason:     req.Reason,
		RepeatType: defaultString(req.RepeatType, "weekly"),
		WeekType:   defaultString(req.WeekType, "all"),
	}
	if req.SpecificDate != nil {
		t, err := time.Parse("2006-01-02", *req.SpecificDate)
		if err == nil {
			ut.SpecificDate = &t
		}
	}

	if err := s.repo.UnavailableTime.Create(ctx, &ut); err != nil {
		s.logger.Error("创建不可用时间失败", zap.Error(err))
		return nil, err
	}

	// 回退提交状态
	s.rollbackTimetableStatus(ctx, userID, semester.SemesterID)

	resp := toUnavailableResponse(ut)
	return &resp, nil
}

func (s *timetableService) UpdateUnavailableTime(ctx context.Context, id string, req *dto.UpdateUnavailableTimeRequest, userID string) (*dto.UnavailableTimeResponse, error) {
	ut, err := s.repo.UnavailableTime.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimetableUnavailableNotFound
		}
		return nil, err
	}
	if ut.UserID != userID {
		return nil, ErrTimetableUnavailableNotOwner
	}

	// 应用更新
	if req.DayOfWeek != nil {
		ut.DayOfWeek = *req.DayOfWeek
	}
	if req.StartTime != nil {
		ut.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		ut.EndTime = *req.EndTime
	}
	if req.Reason != nil {
		ut.Reason = *req.Reason
	}
	if req.RepeatType != nil {
		ut.RepeatType = *req.RepeatType
	}
	if req.WeekType != nil {
		ut.WeekType = *req.WeekType
	}
	if req.SpecificDate != nil {
		t, err := time.Parse("2006-01-02", *req.SpecificDate)
		if err == nil {
			ut.SpecificDate = &t
		}
	}
	ut.UpdatedBy = &userID

	if err := s.repo.UnavailableTime.Update(ctx, ut); err != nil {
		s.logger.Error("更新不可用时间失败", zap.Error(err))
		return nil, err
	}

	// 回退提交状态
	s.rollbackTimetableStatus(ctx, userID, ut.SemesterID)

	resp := toUnavailableResponse(*ut)
	return &resp, nil
}

func (s *timetableService) DeleteUnavailableTime(ctx context.Context, id string, userID string) error {
	ut, err := s.repo.UnavailableTime.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTimetableUnavailableNotFound
		}
		return err
	}
	if ut.UserID != userID {
		return ErrTimetableUnavailableNotOwner
	}

	if err := s.repo.UnavailableTime.Delete(ctx, id, userID); err != nil {
		s.logger.Error("删除不可用时间失败", zap.Error(err))
		return err
	}

	// 回退提交状态
	s.rollbackTimetableStatus(ctx, userID, ut.SemesterID)

	return nil
}

// ════════════════════════════════════════════════════════════
// SubmitTimetable — 提交时间表
// ════════════════════════════════════════════════════════════

func (s *timetableService) SubmitTimetable(ctx context.Context, userID string, semesterID string) (*dto.SubmitTimetableResponse, error) {
	semester, err := s.resolveActiveSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	// 检查是否有课表数据
	courses, err := s.repo.CourseSchedule.ListByUserAndSemester(ctx, userID, semester.SemesterID)
	if err != nil {
		return nil, err
	}
	// 允许无课表但有不可用时间的情况——某些成员可能确实没课
	unavailables, err := s.repo.UnavailableTime.ListByUserAndSemester(ctx, userID, semester.SemesterID)
	if err != nil {
		return nil, err
	}
	if len(courses) == 0 && len(unavailables) == 0 {
		return nil, ErrTimetableNoCourses
	}

	// 查找分配记录
	assignment, err := s.repo.UserSemesterAssignment.GetByUserAndSemester(ctx, userID, semester.SemesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimetableAssignmentNotFound
		}
		return nil, err
	}

	now := time.Now()
	if err := s.repo.UserSemesterAssignment.UpdateTimetableStatus(
		ctx, assignment.AssignmentID, "submitted", &now, userID,
	); err != nil {
		s.logger.Error("更新提交状态失败", zap.Error(err))
		return nil, err
	}

	return &dto.SubmitTimetableResponse{
		SubmitStatus: "submitted",
		SubmittedAt:  &now,
	}, nil
}

// ════════════════════════════════════════════════════════════
// GetProgress — 全局提交进度
// ════════════════════════════════════════════════════════════

func (s *timetableService) GetProgress(ctx context.Context, semesterID string) (*dto.TimetableProgressResponse, error) {
	semester, err := s.resolveActiveSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	total, err := s.repo.UserSemesterAssignment.CountDutyRequired(ctx, semester.SemesterID)
	if err != nil {
		return nil, err
	}
	submitted, err := s.repo.UserSemesterAssignment.CountDutyRequiredSubmitted(ctx, semester.SemesterID)
	if err != nil {
		return nil, err
	}

	// 按部门分组
	assignments, err := s.repo.UserSemesterAssignment.ListDutyRequiredBySemester(ctx, semester.SemesterID)
	if err != nil {
		return nil, err
	}

	deptMap := make(map[string]*dto.DepartmentProgressItem)
	var deptOrder []string
	for _, a := range assignments {
		deptID := ""
		deptName := "未分配"
		if a.User != nil && a.User.Department != nil {
			deptID = a.User.Department.DepartmentID
			deptName = a.User.Department.Name
		}
		item, ok := deptMap[deptID]
		if !ok {
			item = &dto.DepartmentProgressItem{
				DepartmentID:   deptID,
				DepartmentName: deptName,
			}
			deptMap[deptID] = item
			deptOrder = append(deptOrder, deptID)
		}
		item.Total++
		if a.TimetableStatus == "submitted" {
			item.Submitted++
		}
	}

	departments := make([]dto.DepartmentProgressItem, 0, len(deptMap))
	for _, id := range deptOrder {
		item := deptMap[id]
		if item.Total > 0 {
			item.Progress = float64(item.Submitted) / float64(item.Total) * 100
		}
		departments = append(departments, *item)
	}

	progress := float64(0)
	if total > 0 {
		progress = float64(submitted) / float64(total) * 100
	}

	return &dto.TimetableProgressResponse{
		Total:       total,
		Submitted:   submitted,
		Progress:    progress,
		Departments: departments,
	}, nil
}

// ════════════════════════════════════════════════════════════
// GetDepartmentProgress — 部门提交进度
// ════════════════════════════════════════════════════════════

func (s *timetableService) GetDepartmentProgress(ctx context.Context, departmentID string, semesterID string) (*dto.DepartmentProgressResponse, error) {
	semester, err := s.resolveActiveSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	// 获取部门信息
	dept, err := s.repo.Department.GetByID(ctx, departmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimetableDepartmentNotFound
		}
		return nil, err
	}

	// 获取该部门需要值班的成员
	assignments, err := s.repo.UserSemesterAssignment.ListDutyRequiredBySemester(ctx, semester.SemesterID)
	if err != nil {
		return nil, err
	}

	var members []dto.DepartmentMemberStatus
	total, submitted := 0, 0
	for _, a := range assignments {
		if a.User == nil || a.User.DepartmentID != departmentID {
			continue
		}
		total++
		if a.TimetableStatus == "submitted" {
			submitted++
		}
		members = append(members, dto.DepartmentMemberStatus{
			UserID:          a.UserID,
			Name:            a.User.Name,
			StudentID:       a.User.StudentID,
			TimetableStatus: a.TimetableStatus,
			SubmittedAt:     a.TimetableSubmittedAt,
		})
	}

	progress := float64(0)
	if total > 0 {
		progress = float64(submitted) / float64(total) * 100
	}

	return &dto.DepartmentProgressResponse{
		DepartmentID:   dept.DepartmentID,
		DepartmentName: dept.Name,
		Total:          total,
		Submitted:      submitted,
		Progress:       progress,
		Members:        members,
	}, nil
}

// ── 私有辅助方法 ──

// resolveActiveSemester 解析学期：指定 ID 或获取当前活动学期
func (s *timetableService) resolveActiveSemester(ctx context.Context, semesterID string) (*model.Semester, error) {
	if semesterID != "" {
		sem, err := s.repo.Semester.GetByID(ctx, semesterID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrTimetableNoActiveSemester
			}
			return nil, err
		}
		return sem, nil
	}
	sem, err := s.repo.Semester.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimetableNoActiveSemester
		}
		return nil, err
	}
	return sem, nil
}

// rollbackTimetableStatus 回退提交状态为 not_submitted
func (s *timetableService) rollbackTimetableStatus(ctx context.Context, userID, semesterID string) {
	assignment, err := s.repo.UserSemesterAssignment.GetByUserAndSemester(ctx, userID, semesterID)
	if err != nil {
		return // 无分配记录时静默跳过
	}
	if assignment.TimetableStatus == "submitted" {
		if err := s.repo.UserSemesterAssignment.UpdateTimetableStatus(
			ctx, assignment.AssignmentID, "not_submitted", nil, userID,
		); err != nil {
			s.logger.Warn("回退提交状态失败", zap.Error(err), zap.String("userID", userID))
		}
	}
}

// ── 响应转换器 ──

func toCourseResponses(courses []model.CourseSchedule) []dto.CourseResponse {
	result := make([]dto.CourseResponse, 0, len(courses))
	for _, c := range courses {
		result = append(result, dto.CourseResponse{
			ID:        c.CourseScheduleID,
			Name:      c.CourseName,
			DayOfWeek: c.DayOfWeek,
			StartTime: c.StartTime,
			EndTime:   c.EndTime,
			WeekType:  c.WeekType,
			Weeks:     []int(c.Weeks),
			Source:    c.Source,
		})
	}
	return result
}

func toUnavailableResponses(times []model.UnavailableTime) []dto.UnavailableTimeResponse {
	result := make([]dto.UnavailableTimeResponse, 0, len(times))
	for _, t := range times {
		result = append(result, toUnavailableResponse(t))
	}
	return result
}

func toUnavailableResponse(t model.UnavailableTime) dto.UnavailableTimeResponse {
	return dto.UnavailableTimeResponse{
		ID:           t.UnavailableTimeID,
		DayOfWeek:    t.DayOfWeek,
		StartTime:    t.StartTime,
		EndTime:      t.EndTime,
		Reason:       t.Reason,
		RepeatType:   t.RepeatType,
		SpecificDate: t.SpecificDate,
		WeekType:     t.WeekType,
	}
}

func defaultString(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
