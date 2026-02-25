package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 排班模块业务错误 ──

var (
	ErrScheduleNotFound         = errors.New("排班表不存在")
	ErrScheduleItemNotFound     = errors.New("排班项不存在")
	ErrScheduleAlreadyExists    = errors.New("该学期已存在排班表")
	ErrScheduleNotDraft         = errors.New("排班表非草稿状态，不可执行此操作")
	ErrScheduleNotPublished     = errors.New("排班表非已发布状态")
	ErrScheduleCannotPublish    = errors.New("排班表不可发布")
	ErrSubmissionRateIncomplete = errors.New("课表提交率未达100%")
	ErrNoEligibleMembers        = errors.New("无符合条件的排班候选人")
	ErrNoActiveTimeSlots        = errors.New("无可用时间段")
	ErrCandidateNotAvailable    = errors.New("候选人在该时段不可用")
)

// ScheduleService 排班业务接口
type ScheduleService interface {
	// 自动排班
	AutoSchedule(ctx context.Context, req *dto.AutoScheduleRequest, callerID string) (*dto.AutoScheduleResponse, error)
	// 获取排班表（含明细）
	GetSchedule(ctx context.Context, semesterID string) (*dto.ScheduleResponse, error)
	// 获取我的排班
	GetMySchedule(ctx context.Context, semesterID, userID string) ([]dto.ScheduleItemResponse, error)
	// 手动调整排班项（草稿状态）
	UpdateItem(ctx context.Context, itemID string, req *dto.UpdateScheduleItemRequest, callerID string) (*dto.ScheduleItemResponse, error)
	// 校验候选人
	ValidateCandidate(ctx context.Context, itemID string, req *dto.ValidateCandidateRequest) (*dto.ValidateCandidateResponse, error)
	// 获取时段可用候选人
	GetCandidates(ctx context.Context, itemID string) ([]dto.CandidateResponse, error)
	// 发布排班表
	Publish(ctx context.Context, req *dto.PublishScheduleRequest, callerID string) (*dto.ScheduleResponse, error)
	// 发布后修改排班项
	UpdatePublishedItem(ctx context.Context, itemID string, req *dto.UpdatePublishedItemRequest, callerID string) (*dto.ScheduleItemResponse, error)
	// 获取变更日志
	ListChangeLogs(ctx context.Context, req *dto.ScheduleChangeLogListRequest) ([]dto.ScheduleChangeLogResponse, int64, error)
	// 范围检测
	CheckScope(ctx context.Context, scheduleID string) (*dto.ScopeCheckResponse, error)
}

type scheduleService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewScheduleService 创建 ScheduleService 实例
func NewScheduleService(repo *repository.Repository, logger *zap.Logger) ScheduleService {
	return &scheduleService{repo: repo, logger: logger}
}

// ════════════════════════════════════════════════════════════
// AutoSchedule — 4 阶段贪心排班
// ════════════════════════════════════════════════════════════

func (s *scheduleService) AutoSchedule(ctx context.Context, req *dto.AutoScheduleRequest, callerID string) (*dto.AutoScheduleResponse, error) {
	semesterID := req.SemesterID

	// 0. 校验学期
	semester, err := s.repo.Semester.GetByID(ctx, semesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSemesterNotFound
		}
		s.logger.Error("查询学期失败", zap.Error(err))
		return nil, err
	}

	// 0.1 检查是否已有非归档排班表 → 如有则先归档
	existing, err := s.repo.Schedule.GetBySemester(ctx, semesterID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("查询已有排班表失败", zap.Error(err))
		return nil, err
	}
	if existing != nil {
		existing.Status = "archived"
		existing.UpdatedBy = &callerID
		if err := s.repo.Schedule.Update(ctx, existing); err != nil {
			s.logger.Error("归档旧排班表失败", zap.Error(err))
			return nil, err
		}
	}

	// ── 阶段1: 数据准备 ──

	// 1.1 检查课表提交率
	totalRequired, err := s.repo.UserSemesterAssignment.CountDutyRequired(ctx, semesterID)
	if err != nil {
		s.logger.Error("查询需值班人数失败", zap.Error(err))
		return nil, err
	}
	totalSubmitted, err := s.repo.UserSemesterAssignment.CountDutyRequiredSubmitted(ctx, semesterID)
	if err != nil {
		s.logger.Error("查询已提交人数失败", zap.Error(err))
		return nil, err
	}
	if totalRequired == 0 || totalSubmitted < totalRequired {
		return nil, ErrSubmissionRateIncomplete
	}

	// 1.2 获取候选人（duty_required + submitted）
	assignments, err := s.repo.UserSemesterAssignment.ListDutyRequiredSubmitted(ctx, semesterID)
	if err != nil {
		s.logger.Error("查询候选人失败", zap.Error(err))
		return nil, err
	}
	if len(assignments) == 0 {
		return nil, ErrNoEligibleMembers
	}

	// 1.3 获取时间段
	timeSlots, err := s.repo.TimeSlot.List(ctx, semesterID, nil)
	if err != nil {
		s.logger.Error("查询时间段失败", zap.Error(err))
		return nil, err
	}
	if len(timeSlots) == 0 {
		return nil, ErrNoActiveTimeSlots
	}

	// 1.4 获取课表
	courses, err := s.repo.CourseSchedule.ListBySemester(ctx, semesterID)
	if err != nil {
		s.logger.Error("查询课表失败", zap.Error(err))
		return nil, err
	}

	// 1.5 获取不可用时间
	unavailables, err := s.repo.UnavailableTime.ListBySemester(ctx, semesterID)
	if err != nil {
		s.logger.Error("查询不可用时间失败", zap.Error(err))
		return nil, err
	}

	// 1.6 获取排班规则
	rules, err := s.repo.ScheduleRule.List(ctx)
	if err != nil {
		s.logger.Error("查询排班规则失败", zap.Error(err))
		return nil, err
	}
	rulesMap := make(map[string]bool)
	for _, r := range rules {
		rulesMap[r.RuleCode] = r.IsEnabled
	}

	// ── 阶段2: 可用性矩阵构建 ──
	// key: "userID:weekNumber:timeSlotID" → bool（true=可用）
	// 同时记录冲突原因

	// 构建用户课表索引: userID → []CourseSchedule
	userCourses := make(map[string][]model.CourseSchedule)
	for _, c := range courses {
		userCourses[c.UserID] = append(userCourses[c.UserID], c)
	}

	// 构建用户不可用时间索引: userID → []UnavailableTime
	userUnavailables := make(map[string][]model.UnavailableTime)
	for _, u := range unavailables {
		userUnavailables[u.UserID] = append(userUnavailables[u.UserID], u)
	}

	// 候选人列表
	type candidate struct {
		userID       string
		departmentID string
		name         string
	}
	candidates := make([]candidate, 0, len(assignments))
	for _, a := range assignments {
		if a.User != nil {
			candidates = append(candidates, candidate{
				userID:       a.UserID,
				departmentID: a.User.DepartmentID,
				name:         a.User.Name,
			})
		}
	}

	// 排班槽位: week_number × time_slot
	type slot struct {
		weekNumber int
		timeSlot   model.TimeSlot
	}
	var slots []slot
	for _, ts := range timeSlots {
		slots = append(slots, slot{weekNumber: 1, timeSlot: ts})
		slots = append(slots, slot{weekNumber: 2, timeSlot: ts})
	}

	// 可用性矩阵
	type availability struct {
		available bool
		conflicts []string
	}
	availMatrix := make(map[string]*availability) // "userID:week:slotID"

	for _, c := range candidates {
		for _, sl := range slots {
			key := fmt.Sprintf("%s:%d:%s", c.userID, sl.weekNumber, sl.timeSlot.TimeSlotID)
			avail := &availability{available: true}

			weekType := weekNumberToType(sl.weekNumber, semester.FirstWeekType)

			// R1: 课表冲突检测（硬约束）
			if rulesMap["R1"] {
				for _, course := range userCourses[c.userID] {
					if hasTimeConflict(course.DayOfWeek, course.StartTime, course.EndTime, course.WeekType,
						sl.timeSlot.DayOfWeek, sl.timeSlot.StartTime, sl.timeSlot.EndTime, weekType) {
						avail.available = false
						avail.conflicts = append(avail.conflicts, fmt.Sprintf("课程冲突: %s", course.CourseName))
					}
				}
			}

			// R2: 不可用时间检测（硬约束）
			if rulesMap["R2"] {
				for _, ut := range userUnavailables[c.userID] {
					if hasUnavailableConflict(ut, sl.timeSlot.DayOfWeek, sl.timeSlot.StartTime, sl.timeSlot.EndTime, weekType) {
						avail.available = false
						reason := "不可用时间冲突"
						if ut.Reason != "" {
							reason = fmt.Sprintf("不可用时间: %s", ut.Reason)
						}
						avail.conflicts = append(avail.conflicts, reason)
					}
				}
			}

			availMatrix[key] = avail
		}
	}

	// ── 阶段3: 贪心排班 ──

	// 统计每个槽位的可用人数
	type slotInfo struct {
		slot           slot
		availableCount int
	}
	slotInfos := make([]slotInfo, 0, len(slots))
	for _, sl := range slots {
		count := 0
		for _, c := range candidates {
			key := fmt.Sprintf("%s:%d:%s", c.userID, sl.weekNumber, sl.timeSlot.TimeSlotID)
			if a, ok := availMatrix[key]; ok && a.available {
				count++
			}
		}
		slotInfos = append(slotInfos, slotInfo{slot: sl, availableCount: count})
	}

	// 按可用人数升序排列（最难排的槽位优先）
	sort.Slice(slotInfos, func(i, j int) bool {
		return slotInfos[i].availableCount < slotInfos[j].availableCount
	})

	// 排班结果
	type assignment struct {
		weekNumber int
		timeSlotID string
		memberID   string
	}
	var result []assignment
	warnings := make([]string, 0)

	// 跟踪每人排班次数
	memberCount := make(map[string]int)
	// 跟踪每人每天已排（R6: 同人同日不重复）
	memberDayWeek := make(map[string]bool) // "userID:week:dayOfWeek"
	// 跟踪每天每部门已排（R3: 同日部门不重复）
	dayDeptWeek := make(map[string]bool) // "week:dayOfWeek:deptID"
	// 跟踪相邻班次部门（R4）
	slotDeptWeek := make(map[string]string) // "week:slotID" → deptID

	for _, si := range slotInfos {
		sl := si.slot
		dayKey := fmt.Sprintf("%d:%d", sl.weekNumber, sl.timeSlot.DayOfWeek)

		// 收集可用候选人
		type scoredCandidate struct {
			candidate candidate
			score     int // 越小越优先
		}
		var availCandidates []scoredCandidate

		for _, c := range candidates {
			key := fmt.Sprintf("%s:%d:%s", c.userID, sl.weekNumber, sl.timeSlot.TimeSlotID)
			avail, ok := availMatrix[key]
			if !ok || !avail.available {
				continue
			}

			// R6: 同人同日不重复（硬约束）
			mdKey := fmt.Sprintf("%s:%s", c.userID, dayKey)
			if memberDayWeek[mdKey] {
				continue
			}

			score := memberCount[c.userID] * 100 // 当前排班少优先

			// R3: 同日部门不重复（软约束）
			if rulesMap["R3"] {
				ddKey := fmt.Sprintf("%s:%s", dayKey, c.departmentID)
				if dayDeptWeek[ddKey] {
					score += 50
				}
			}

			// R4: 相邻班次部门不重复（软约束）
			if rulesMap["R4"] {
				slotKey := fmt.Sprintf("%d:%s", sl.weekNumber, sl.timeSlot.TimeSlotID)
				if prevDept, exists := slotDeptWeek[slotKey]; exists && prevDept == c.departmentID {
					score += 30
				}
			}

			// R5: 单双周早八不重复（软约束）
			if rulesMap["R5"] {
				if sl.timeSlot.StartTime <= "08:30" {
					otherWeek := 3 - sl.weekNumber // 1→2, 2→1
					for _, otherSl := range slots {
						if otherSl.weekNumber == otherWeek &&
							otherSl.timeSlot.DayOfWeek == sl.timeSlot.DayOfWeek &&
							otherSl.timeSlot.StartTime <= "08:30" {
							otherKey := fmt.Sprintf("%d:%s", otherWeek, otherSl.timeSlot.TimeSlotID)
							if assignedDept, exists := slotDeptWeek[otherKey]; exists && assignedDept == c.departmentID {
								score += 20
							}
						}
					}
				}
			}

			availCandidates = append(availCandidates, scoredCandidate{candidate: c, score: score})
		}

		if len(availCandidates) == 0 {
			warnings = append(warnings, fmt.Sprintf("时段 %s (第%d周 周%d %s-%s) 无可用候选人",
				sl.timeSlot.Name, sl.weekNumber, sl.timeSlot.DayOfWeek, sl.timeSlot.StartTime, sl.timeSlot.EndTime))
			continue
		}

		// 按分数排序，分数相同则随机（通过名字排序保证稳定性）
		sort.Slice(availCandidates, func(i, j int) bool {
			if availCandidates[i].score != availCandidates[j].score {
				return availCandidates[i].score < availCandidates[j].score
			}
			return availCandidates[i].candidate.name < availCandidates[j].candidate.name
		})

		chosen := availCandidates[0].candidate
		result = append(result, assignment{
			weekNumber: sl.weekNumber,
			timeSlotID: sl.timeSlot.TimeSlotID,
			memberID:   chosen.userID,
		})

		// 更新跟踪状态
		memberCount[chosen.userID]++
		memberDayWeek[fmt.Sprintf("%s:%s", chosen.userID, dayKey)] = true
		dayDeptWeek[fmt.Sprintf("%s:%s", dayKey, chosen.departmentID)] = true
		slotDeptWeek[fmt.Sprintf("%d:%s", sl.weekNumber, sl.timeSlot.TimeSlotID)] = chosen.departmentID
	}

	// ── 阶段4: 输出 ──

	// 创建排班表
	schedule := &model.Schedule{
		SemesterID: semesterID,
		Status:     "draft",
	}
	schedule.CreatedBy = &callerID
	schedule.UpdatedBy = &callerID

	if err := s.repo.Schedule.Create(ctx, schedule); err != nil {
		s.logger.Error("创建排班表失败", zap.Error(err))
		return nil, err
	}

	// 批量创建排班项
	items := make([]model.ScheduleItem, 0, len(result))
	for _, r := range result {
		item := model.ScheduleItem{
			ScheduleID: schedule.ScheduleID,
			WeekNumber: r.weekNumber,
			TimeSlotID: r.timeSlotID,
			MemberID:   r.memberID,
		}
		item.CreatedBy = &callerID
		item.UpdatedBy = &callerID
		items = append(items, item)
	}

	if len(items) > 0 {
		if err := s.repo.ScheduleItem.BatchCreate(ctx, items); err != nil {
			s.logger.Error("批量创建排班项失败", zap.Error(err))
			return nil, err
		}
	}

	// 保存成员快照
	snapshots := make([]model.ScheduleMemberSnapshot, 0, len(candidates))
	now := time.Now()
	for _, c := range candidates {
		snapshots = append(snapshots, model.ScheduleMemberSnapshot{
			ScheduleID:   schedule.ScheduleID,
			UserID:       c.userID,
			DepartmentID: c.departmentID,
			SnapshotAt:   now,
			CreatedAt:    now,
		})
	}
	if len(snapshots) > 0 {
		if err := s.repo.ScheduleMemberSnapshot.BatchCreate(ctx, snapshots); err != nil {
			s.logger.Error("保存成员快照失败", zap.Error(err))
			return nil, err
		}
	}

	// 构建响应
	scheduleResp, err := s.buildScheduleResponse(ctx, schedule)
	if err != nil {
		s.logger.Error("构建排班响应失败", zap.Error(err))
		return nil, err
	}

	return &dto.AutoScheduleResponse{
		Schedule:    scheduleResp,
		TotalSlots:  len(slots),
		FilledSlots: len(result),
		Warnings:    warnings,
	}, nil
}

// ════════════════════════════════════════════════════════════
// GetSchedule — 获取排班表
// ════════════════════════════════════════════════════════════

func (s *scheduleService) GetSchedule(ctx context.Context, semesterID string) (*dto.ScheduleResponse, error) {
	schedule, err := s.repo.Schedule.GetBySemester(ctx, semesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		s.logger.Error("查询排班表失败", zap.Error(err))
		return nil, err
	}

	return s.buildScheduleResponse(ctx, schedule)
}

// ════════════════════════════════════════════════════════════
// GetMySchedule — 获取我的排班
// ════════════════════════════════════════════════════════════

func (s *scheduleService) GetMySchedule(ctx context.Context, semesterID, userID string) ([]dto.ScheduleItemResponse, error) {
	schedule, err := s.repo.Schedule.GetBySemester(ctx, semesterID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		s.logger.Error("查询排班表失败", zap.Error(err))
		return nil, err
	}

	items, err := s.repo.ScheduleItem.ListByScheduleAndMember(ctx, schedule.ScheduleID, userID)
	if err != nil {
		s.logger.Error("查询我的排班失败", zap.Error(err))
		return nil, err
	}

	result := make([]dto.ScheduleItemResponse, 0, len(items))
	for i := range items {
		result = append(result, s.toScheduleItemResponse(&items[i]))
	}
	return result, nil
}

// ════════════════════════════════════════════════════════════
// UpdateItem — 手动调整排班项（draft 状态）
// ════════════════════════════════════════════════════════════

func (s *scheduleService) UpdateItem(ctx context.Context, itemID string, req *dto.UpdateScheduleItemRequest, callerID string) (*dto.ScheduleItemResponse, error) {
	item, err := s.repo.ScheduleItem.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleItemNotFound
		}
		s.logger.Error("查询排班项失败", zap.Error(err))
		return nil, err
	}

	// 检查排班表状态
	schedule, err := s.repo.Schedule.GetByID(ctx, item.ScheduleID)
	if err != nil {
		s.logger.Error("查询排班表失败", zap.Error(err))
		return nil, err
	}
	if schedule.Status != "draft" {
		return nil, ErrScheduleNotDraft
	}

	if req.MemberID != nil {
		item.MemberID = *req.MemberID
	}
	if req.LocationID != nil {
		item.LocationID = req.LocationID
	}
	item.UpdatedBy = &callerID

	if err := s.repo.ScheduleItem.Update(ctx, item); err != nil {
		s.logger.Error("更新排班项失败", zap.Error(err))
		return nil, err
	}

	// 重新查询以获取完整关联
	updated, err := s.repo.ScheduleItem.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}

	resp := s.toScheduleItemResponse(updated)
	return &resp, nil
}

// ════════════════════════════════════════════════════════════
// ValidateCandidate — 校验候选人
// ════════════════════════════════════════════════════════════

func (s *scheduleService) ValidateCandidate(ctx context.Context, itemID string, req *dto.ValidateCandidateRequest) (*dto.ValidateCandidateResponse, error) {
	item, err := s.repo.ScheduleItem.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleItemNotFound
		}
		return nil, err
	}

	schedule, err := s.repo.Schedule.GetByID(ctx, item.ScheduleID)
	if err != nil {
		return nil, err
	}

	conflicts := s.checkCandidateConflicts(ctx, req.MemberID, schedule.SemesterID, item, schedule)
	return &dto.ValidateCandidateResponse{
		Valid:     len(conflicts) == 0,
		Conflicts: conflicts,
	}, nil
}

// ════════════════════════════════════════════════════════════
// GetCandidates — 获取时段可用候选人
// ════════════════════════════════════════════════════════════

func (s *scheduleService) GetCandidates(ctx context.Context, itemID string) ([]dto.CandidateResponse, error) {
	item, err := s.repo.ScheduleItem.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleItemNotFound
		}
		return nil, err
	}

	schedule, err := s.repo.Schedule.GetByID(ctx, item.ScheduleID)
	if err != nil {
		return nil, err
	}

	// 获取候选人
	assignments, err := s.repo.UserSemesterAssignment.ListDutyRequiredSubmitted(ctx, schedule.SemesterID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.CandidateResponse, 0, len(assignments))
	for _, a := range assignments {
		if a.User == nil {
			continue
		}
		conflicts := s.checkCandidateConflicts(ctx, a.UserID, schedule.SemesterID, item, schedule)
		cr := dto.CandidateResponse{
			UserID:    a.UserID,
			Name:      a.User.Name,
			StudentID: a.User.StudentID,
			Available: len(conflicts) == 0,
			Conflicts: conflicts,
		}
		if a.User.Department != nil {
			cr.Department = &dto.DepartmentResponse{
				ID:   a.User.Department.DepartmentID,
				Name: a.User.Department.Name,
			}
		}
		result = append(result, cr)
	}

	return result, nil
}

// ════════════════════════════════════════════════════════════
// Publish — 发布排班表
// ════════════════════════════════════════════════════════════

func (s *scheduleService) Publish(ctx context.Context, req *dto.PublishScheduleRequest, callerID string) (*dto.ScheduleResponse, error) {
	schedule, err := s.repo.Schedule.GetByID(ctx, req.ScheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}

	if schedule.Status != "draft" && schedule.Status != "need_regen" {
		return nil, ErrScheduleCannotPublish
	}

	now := time.Now()
	schedule.Status = "published"
	schedule.PublishedAt = &now
	schedule.UpdatedBy = &callerID

	if err := s.repo.Schedule.Update(ctx, schedule); err != nil {
		s.logger.Error("发布排班表失败", zap.Error(err))
		return nil, err
	}

	return s.buildScheduleResponse(ctx, schedule)
}

// ════════════════════════════════════════════════════════════
// UpdatePublishedItem — 发布后修改排班项
// ════════════════════════════════════════════════════════════

func (s *scheduleService) UpdatePublishedItem(ctx context.Context, itemID string, req *dto.UpdatePublishedItemRequest, callerID string) (*dto.ScheduleItemResponse, error) {
	item, err := s.repo.ScheduleItem.GetByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleItemNotFound
		}
		return nil, err
	}

	schedule, err := s.repo.Schedule.GetByID(ctx, item.ScheduleID)
	if err != nil {
		return nil, err
	}

	if schedule.Status != "published" {
		return nil, ErrScheduleNotPublished
	}

	// 校验新候选人
	conflicts := s.checkCandidateConflicts(ctx, req.MemberID, schedule.SemesterID, item, schedule)
	if len(conflicts) > 0 {
		return nil, ErrCandidateNotAvailable
	}

	// 记录变更日志
	changeLog := &model.ScheduleChangeLog{
		ScheduleID:       schedule.ScheduleID,
		ScheduleItemID:   item.ScheduleItemID,
		OriginalMemberID: item.MemberID,
		NewMemberID:      req.MemberID,
		ChangeType:       "admin_modify",
		Reason:           req.Reason,
		OperatorID:       callerID,
		CreatedAt:        time.Now(),
	}
	if err := s.repo.ScheduleChangeLog.Create(ctx, changeLog); err != nil {
		s.logger.Error("创建变更日志失败", zap.Error(err))
		return nil, err
	}

	// 更新排班项
	item.MemberID = req.MemberID
	item.UpdatedBy = &callerID
	if err := s.repo.ScheduleItem.Update(ctx, item); err != nil {
		s.logger.Error("更新排班项失败", zap.Error(err))
		return nil, err
	}

	updated, err := s.repo.ScheduleItem.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}

	resp := s.toScheduleItemResponse(updated)
	return &resp, nil
}

// ════════════════════════════════════════════════════════════
// ListChangeLogs — 获取变更日志
// ════════════════════════════════════════════════════════════

func (s *scheduleService) ListChangeLogs(ctx context.Context, req *dto.ScheduleChangeLogListRequest) ([]dto.ScheduleChangeLogResponse, int64, error) {
	logs, total, err := s.repo.ScheduleChangeLog.ListBySchedule(ctx, req.ScheduleID, req.GetOffset(), req.GetPageSize())
	if err != nil {
		s.logger.Error("查询变更日志失败", zap.Error(err))
		return nil, 0, err
	}

	result := make([]dto.ScheduleChangeLogResponse, 0, len(logs))
	for _, l := range logs {
		result = append(result, dto.ScheduleChangeLogResponse{
			ID:               l.ChangeLogID,
			ScheduleID:       l.ScheduleID,
			ScheduleItemID:   l.ScheduleItemID,
			OriginalMemberID: l.OriginalMemberID,
			NewMemberID:      l.NewMemberID,
			ChangeType:       l.ChangeType,
			Reason:           l.Reason,
			OperatorID:       l.OperatorID,
			CreatedAt:        l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return result, total, nil
}

// ════════════════════════════════════════════════════════════
// CheckScope — 范围变更检测
// ════════════════════════════════════════════════════════════

func (s *scheduleService) CheckScope(ctx context.Context, scheduleID string) (*dto.ScopeCheckResponse, error) {
	schedule, err := s.repo.Schedule.GetByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}

	// 获取快照
	snapshots, err := s.repo.ScheduleMemberSnapshot.ListBySchedule(ctx, scheduleID)
	if err != nil {
		return nil, err
	}

	// 获取当前候选人
	assignments, err := s.repo.UserSemesterAssignment.ListDutyRequiredSubmitted(ctx, schedule.SemesterID)
	if err != nil {
		return nil, err
	}

	// 对比
	snapshotUsers := make(map[string]bool)
	for _, s := range snapshots {
		snapshotUsers[s.UserID] = true
	}
	currentUsers := make(map[string]bool)
	currentUserNames := make(map[string]string)
	for _, a := range assignments {
		currentUsers[a.UserID] = true
		if a.User != nil {
			currentUserNames[a.UserID] = a.User.Name
		}
	}

	var added, removed []string
	for uid := range currentUsers {
		if !snapshotUsers[uid] {
			name := currentUserNames[uid]
			if name == "" {
				name = uid
			}
			added = append(added, name)
		}
	}
	for uid := range snapshotUsers {
		if !currentUsers[uid] {
			removed = append(removed, uid)
		}
	}

	changed := len(added) > 0 || len(removed) > 0

	// 如果范围变更且排班表已发布，自动标记为 need_regen
	if changed && schedule.Status == "published" {
		schedule.Status = "need_regen"
		if err := s.repo.Schedule.Update(ctx, schedule); err != nil {
			s.logger.Error("标记need_regen失败", zap.Error(err))
			return nil, err
		}
	}

	return &dto.ScopeCheckResponse{
		Changed:      changed,
		AddedUsers:   added,
		RemovedUsers: removed,
	}, nil
}

// ════════════════════════════════════════════════════════════
// 内部辅助方法
// ════════════════════════════════════════════════════════════

// weekNumberToType 将周数(1,2)转为 odd/even
func weekNumberToType(weekNumber int, firstWeekType string) string {
	if firstWeekType == "odd" {
		if weekNumber == 1 {
			return "odd"
		}
		return "even"
	}
	// firstWeekType == "even"
	if weekNumber == 1 {
		return "even"
	}
	return "odd"
}

// hasTimeConflict 检查时间是否有冲突
func hasTimeConflict(courseDOW int, courseStart, courseEnd, courseWeekType string,
	slotDOW int, slotStart, slotEnd, slotWeekType string) bool {
	// 检查星期
	if courseDOW != slotDOW {
		return false
	}
	// 检查周次类型
	if courseWeekType != "all" && slotWeekType != "all" && courseWeekType != slotWeekType {
		return false
	}
	// 检查时间段重叠
	return slotStart < courseEnd && courseStart < slotEnd
}

// hasUnavailableConflict 检查不可用时间冲突
func hasUnavailableConflict(ut model.UnavailableTime, slotDOW int, slotStart, slotEnd, slotWeekType string) bool {
	// 检查星期
	if ut.DayOfWeek != slotDOW {
		return false
	}
	// 检查周次类型
	if ut.WeekType != "all" && slotWeekType != "all" && ut.WeekType != slotWeekType {
		return false
	}
	// weekly 类型直接检查时间重叠
	// once 类型也检查（简化处理：按模板排班不区分具体日期）
	return slotStart < ut.EndTime && ut.StartTime < slotEnd
}

// checkCandidateConflicts 检查候选人在指定排班项时段的冲突
func (s *scheduleService) checkCandidateConflicts(ctx context.Context, memberID, semesterID string, item *model.ScheduleItem, schedule *model.Schedule) []string {
	var conflicts []string

	// 获取学期信息
	semester, err := s.repo.Semester.GetByID(ctx, semesterID)
	if err != nil {
		return []string{"无法获取学期信息"}
	}

	// 获取规则
	rules, _ := s.repo.ScheduleRule.List(ctx)
	rulesMap := make(map[string]bool)
	for _, r := range rules {
		rulesMap[r.RuleCode] = r.IsEnabled
	}

	weekType := weekNumberToType(item.WeekNumber, semester.FirstWeekType)

	// R1: 课表冲突
	if rulesMap["R1"] {
		courses, _ := s.repo.CourseSchedule.ListByUserAndSemester(ctx, memberID, semesterID)
		if item.TimeSlot != nil {
			for _, c := range courses {
				if hasTimeConflict(c.DayOfWeek, c.StartTime, c.EndTime, c.WeekType,
					item.TimeSlot.DayOfWeek, item.TimeSlot.StartTime, item.TimeSlot.EndTime, weekType) {
					conflicts = append(conflicts, fmt.Sprintf("课程冲突: %s", c.CourseName))
				}
			}
		}
	}

	// R2: 不可用时间
	if rulesMap["R2"] {
		unavailables, _ := s.repo.UnavailableTime.ListByUserAndSemester(ctx, memberID, semesterID)
		if item.TimeSlot != nil {
			for _, ut := range unavailables {
				if hasUnavailableConflict(ut, item.TimeSlot.DayOfWeek, item.TimeSlot.StartTime, item.TimeSlot.EndTime, weekType) {
					reason := "不可用时间冲突"
					if ut.Reason != "" {
						reason = fmt.Sprintf("不可用时间: %s", ut.Reason)
					}
					conflicts = append(conflicts, reason)
				}
			}
		}
	}

	// R6: 同人同日不重复
	allItems, _ := s.repo.ScheduleItem.ListBySchedule(ctx, schedule.ScheduleID)
	if item.TimeSlot != nil {
		for _, other := range allItems {
			if other.ScheduleItemID == item.ScheduleItemID {
				continue
			}
			if other.MemberID == memberID && other.WeekNumber == item.WeekNumber && other.TimeSlot != nil &&
				other.TimeSlot.DayOfWeek == item.TimeSlot.DayOfWeek {
				conflicts = append(conflicts, "同人同日重复排班")
				break
			}
		}
	}

	return conflicts
}

// buildScheduleResponse 构建排班表完整响应
func (s *scheduleService) buildScheduleResponse(ctx context.Context, schedule *model.Schedule) (*dto.ScheduleResponse, error) {
	items, err := s.repo.ScheduleItem.ListBySchedule(ctx, schedule.ScheduleID)
	if err != nil {
		return nil, err
	}

	resp := &dto.ScheduleResponse{
		ID:         schedule.ScheduleID,
		SemesterID: schedule.SemesterID,
		Status:     schedule.Status,
		CreatedAt:  schedule.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  schedule.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if schedule.PublishedAt != nil {
		t := schedule.PublishedAt.Format("2006-01-02T15:04:05Z")
		resp.PublishedAt = &t
	}

	if schedule.Semester != nil {
		resp.Semester = &dto.SemesterBrief{
			ID:   schedule.Semester.SemesterID,
			Name: schedule.Semester.Name,
		}
	}

	resp.Items = make([]dto.ScheduleItemResponse, 0, len(items))
	for i := range items {
		resp.Items = append(resp.Items, s.toScheduleItemResponse(&items[i]))
	}

	return resp, nil
}

// toScheduleItemResponse 转换排班项为响应
func (s *scheduleService) toScheduleItemResponse(item *model.ScheduleItem) dto.ScheduleItemResponse {
	resp := dto.ScheduleItemResponse{
		ID:         item.ScheduleItemID,
		ScheduleID: item.ScheduleID,
		WeekNumber: item.WeekNumber,
		CreatedAt:  item.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  item.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if item.TimeSlot != nil {
		resp.TimeSlot = &dto.TimeSlotBrief{
			ID:        item.TimeSlot.TimeSlotID,
			Name:      item.TimeSlot.Name,
			DayOfWeek: item.TimeSlot.DayOfWeek,
			StartTime: item.TimeSlot.StartTime,
			EndTime:   item.TimeSlot.EndTime,
		}
	}

	if item.Member != nil {
		resp.Member = &dto.MemberBrief{
			ID:        item.Member.UserID,
			Name:      item.Member.Name,
			StudentID: item.Member.StudentID,
		}
		if item.Member.Department != nil {
			resp.Member.Department = &dto.DepartmentResponse{
				ID:   item.Member.Department.DepartmentID,
				Name: item.Member.Department.Name,
			}
		}
	}

	if item.Location != nil {
		resp.Location = &dto.LocationBrief{
			ID:   item.Location.LocationID,
			Name: item.Location.Name,
		}
	}

	return resp
}
