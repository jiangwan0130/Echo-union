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

func setupTestScheduleService() (ScheduleService, *testScheduleRepos) {
	repos := newTestScheduleRepos()
	repoAgg := repos.toRepository()
	logger := zap.NewNop()
	svc := NewScheduleService(repoAgg, logger)
	return svc, repos
}

// testScheduleRepos 聚合所有 mock repo 便于 seed 数据
type testScheduleRepos struct {
	semester       *mockSemesterRepo
	timeSlot       *mockTimeSlotRepo
	scheduleRule   *mockScheduleRuleRepo
	courseSchedule *mockCourseScheduleRepo
	unavailable    *mockUnavailableTimeRepo
	assignment     *mockUserSemesterAssignmentRepo
	schedule       *mockScheduleRepo
	scheduleItem   *mockScheduleItemRepo
	snapshot       *mockScheduleMemberSnapshotRepo
	changeLog      *mockScheduleChangeLogRepo
}

func newTestScheduleRepos() *testScheduleRepos {
	return &testScheduleRepos{
		semester:       newMockSemesterRepo(),
		timeSlot:       newMockTimeSlotRepo(),
		scheduleRule:   newMockScheduleRuleRepo(),
		courseSchedule: newMockCourseScheduleRepo(),
		unavailable:    newMockUnavailableTimeRepo(),
		assignment:     newMockUserSemesterAssignmentRepo(),
		schedule:       newMockScheduleRepo(),
		scheduleItem:   newMockScheduleItemRepo(),
		snapshot:       newMockScheduleMemberSnapshotRepo(),
		changeLog:      newMockScheduleChangeLogRepo(),
	}
}

func (r *testScheduleRepos) toRepository() *repository.Repository {
	return &repository.Repository{
		User:                   newMockUserRepo(),
		Department:             newMockDeptRepo(),
		Semester:               r.semester,
		TimeSlot:               r.timeSlot,
		Location:               newMockLocationRepo(),
		SystemConfig:           newMockSystemConfigRepo(),
		ScheduleRule:           r.scheduleRule,
		CourseSchedule:         r.courseSchedule,
		UnavailableTime:        r.unavailable,
		UserSemesterAssignment: r.assignment,
		Schedule:               r.schedule,
		ScheduleItem:           r.scheduleItem,
		ScheduleMemberSnapshot: r.snapshot,
		ScheduleChangeLog:      r.changeLog,
	}
}

// seedBasicData 种子数据：1个学期 + 2个时间段 + 2个候选人 + 规则全开
func seedBasicData(repos *testScheduleRepos) {
	// 学期
	repos.semester.semesters["sem-1"] = &model.Semester{
		SemesterID:    "sem-1",
		Name:          "2025-2026秋",
		IsActive:      true,
		FirstWeekType: "odd",
	}

	// 时间段（2个 = 周一上午/下午）
	semID := "sem-1"
	repos.timeSlot.slots["ts-1"] = &model.TimeSlot{
		TimeSlotID: "ts-1", Name: "周一上午", SemesterID: &semID,
		DayOfWeek: 1, StartTime: "08:10", EndTime: "10:05", IsActive: true,
	}
	repos.timeSlot.slots["ts-2"] = &model.TimeSlot{
		TimeSlotID: "ts-2", Name: "周一下午", SemesterID: &semID,
		DayOfWeek: 1, StartTime: "14:00", EndTime: "16:00", IsActive: true,
	}

	// 候选人
	dept1 := &model.Department{DepartmentID: "dept-1", Name: "技术部"}
	dept2 := &model.Department{DepartmentID: "dept-2", Name: "运营部"}
	user1 := &model.User{UserID: "user-1", Name: "张三", StudentID: "2021001", DepartmentID: "dept-1", Department: dept1}
	user2 := &model.User{UserID: "user-2", Name: "李四", StudentID: "2021002", DepartmentID: "dept-2", Department: dept2}

	repos.assignment.assignments = []model.UserSemesterAssignment{
		{AssignmentID: "a-1", UserID: "user-1", SemesterID: "sem-1", DutyRequired: true, TimetableStatus: "submitted", User: user1},
		{AssignmentID: "a-2", UserID: "user-2", SemesterID: "sem-1", DutyRequired: true, TimetableStatus: "submitted", User: user2},
	}

	// 规则全部启用
	repos.scheduleRule.rules["r1"] = &model.ScheduleRule{RuleID: "r1", RuleCode: "R1", IsEnabled: true}
	repos.scheduleRule.rules["r2"] = &model.ScheduleRule{RuleID: "r2", RuleCode: "R2", IsEnabled: true}
	repos.scheduleRule.rules["r3"] = &model.ScheduleRule{RuleID: "r3", RuleCode: "R3", IsEnabled: true}
	repos.scheduleRule.rules["r4"] = &model.ScheduleRule{RuleID: "r4", RuleCode: "R4", IsEnabled: true}
	repos.scheduleRule.rules["r5"] = &model.ScheduleRule{RuleID: "r5", RuleCode: "R5", IsEnabled: true}
	repos.scheduleRule.rules["r6"] = &model.ScheduleRule{RuleID: "r6", RuleCode: "R6", IsEnabled: true}
}

// ════════════════════════════════════════════════════════════
// AutoSchedule 测试
// ════════════════════════════════════════════════════════════

func TestScheduleService_AutoSchedule_Success(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	req := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	result, err := svc.AutoSchedule(context.Background(), req, "admin-1")
	if err != nil {
		t.Fatalf("AutoSchedule 应成功: %v", err)
	}

	if result.Schedule == nil {
		t.Fatal("Schedule 不应为 nil")
	}
	if result.Schedule.Status != "draft" {
		t.Errorf("期望 status=draft，实际=%s", result.Schedule.Status)
	}
	// 2个时间段 × 2周 = 4个槽位
	if result.TotalSlots != 4 {
		t.Errorf("期望 TotalSlots=4，实际=%d", result.TotalSlots)
	}
	if result.FilledSlots == 0 {
		t.Error("FilledSlots 不应为0")
	}
	if len(result.Schedule.Items) == 0 {
		t.Error("排班项不应为空")
	}
}

func TestScheduleService_AutoSchedule_SemesterNotFound(t *testing.T) {
	svc, _ := setupTestScheduleService()

	req := &dto.AutoScheduleRequest{SemesterID: "nonexistent"}
	_, err := svc.AutoSchedule(context.Background(), req, "admin-1")
	if !errors.Is(err, ErrSemesterNotFound) {
		t.Errorf("期望 ErrSemesterNotFound，实际: %v", err)
	}
}

func TestScheduleService_AutoSchedule_IncompleteSubmission(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 添加一个未提交的候选人
	repos.assignment.assignments = append(repos.assignment.assignments, model.UserSemesterAssignment{
		AssignmentID: "a-3", UserID: "user-3", SemesterID: "sem-1",
		DutyRequired: true, TimetableStatus: "not_submitted",
	})

	req := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	_, err := svc.AutoSchedule(context.Background(), req, "admin-1")
	if !errors.Is(err, ErrSubmissionRateIncomplete) {
		t.Errorf("期望 ErrSubmissionRateIncomplete，实际: %v", err)
	}
}

func TestScheduleService_AutoSchedule_NoTimeSlots(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 清空时间段
	repos.timeSlot.slots = make(map[string]*model.TimeSlot)

	req := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	_, err := svc.AutoSchedule(context.Background(), req, "admin-1")
	if !errors.Is(err, ErrNoActiveTimeSlots) {
		t.Errorf("期望 ErrNoActiveTimeSlots，实际: %v", err)
	}
}

func TestScheduleService_AutoSchedule_CourseConflict(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 给 user-1 添加周一上午的课程冲突
	repos.courseSchedule.courses = []model.CourseSchedule{
		{
			CourseScheduleID: "cs-1", UserID: "user-1", SemesterID: "sem-1",
			CourseName: "高等数学", DayOfWeek: 1,
			StartTime: "08:00", EndTime: "09:50", WeekType: "all",
		},
	}

	req := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	result, err := svc.AutoSchedule(context.Background(), req, "admin-1")
	if err != nil {
		t.Fatalf("AutoSchedule 应成功: %v", err)
	}

	// user-1 不应被排到周一上午（ts-1）的任何周次
	for _, item := range result.Schedule.Items {
		if item.TimeSlot != nil && item.TimeSlot.ID == "ts-1" && item.Member != nil && item.Member.ID == "user-1" {
			t.Errorf("user-1 有课程冲突，不应被排到 ts-1，但被排到了第%d周", item.WeekNumber)
		}
	}
}

func TestScheduleService_AutoSchedule_UnavailableConflict(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 给 user-2 添加周一下午不可用
	repos.unavailable.times = []model.UnavailableTime{
		{
			UnavailableTimeID: "ut-1", UserID: "user-2", SemesterID: "sem-1",
			DayOfWeek: 1, StartTime: "13:00", EndTime: "17:00",
			RepeatType: "weekly", WeekType: "all", Reason: "兼职",
		},
	}

	req := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	result, err := svc.AutoSchedule(context.Background(), req, "admin-1")
	if err != nil {
		t.Fatalf("AutoSchedule 应成功: %v", err)
	}

	// user-2 不应被排到周一下午（ts-2）
	for _, item := range result.Schedule.Items {
		if item.TimeSlot != nil && item.TimeSlot.ID == "ts-2" && item.Member != nil && item.Member.ID == "user-2" {
			t.Error("user-2 有不可用时间，不应被排到 ts-2")
		}
	}
}

func TestScheduleService_AutoSchedule_ArchivesExisting(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 先创建一个旧排班表
	repos.schedule.schedules["old-sched"] = &model.Schedule{
		ScheduleID: "old-sched",
		SemesterID: "sem-1",
		Status:     "draft",
	}

	req := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	_, err := svc.AutoSchedule(context.Background(), req, "admin-1")
	if err != nil {
		t.Fatalf("AutoSchedule 应成功: %v", err)
	}

	// 旧排班表应被归档
	if repos.schedule.schedules["old-sched"].Status != "archived" {
		t.Errorf("旧排班表应被归档，实际status=%s", repos.schedule.schedules["old-sched"].Status)
	}
}

// ════════════════════════════════════════════════════════════
// GetSchedule 测试
// ════════════════════════════════════════════════════════════

func TestScheduleService_GetSchedule_Success(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 先执行自动排班
	req := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	_, _ = svc.AutoSchedule(context.Background(), req, "admin-1")

	result, err := svc.GetSchedule(context.Background(), "sem-1")
	if err != nil {
		t.Fatalf("GetSchedule 应成功: %v", err)
	}
	if result == nil {
		t.Fatal("结果不应为 nil")
	}
	if result.Status != "draft" {
		t.Errorf("期望 status=draft，实际=%s", result.Status)
	}
}

func TestScheduleService_GetSchedule_NotFound(t *testing.T) {
	svc, _ := setupTestScheduleService()

	_, err := svc.GetSchedule(context.Background(), "nonexistent")
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("期望 ErrScheduleNotFound，实际: %v", err)
	}
}

// ════════════════════════════════════════════════════════════
// GetMySchedule 测试
// ════════════════════════════════════════════════════════════

func TestScheduleService_GetMySchedule_Success(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 先执行自动排班
	req := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	_, _ = svc.AutoSchedule(context.Background(), req, "admin-1")

	result, err := svc.GetMySchedule(context.Background(), "sem-1", "user-1")
	if err != nil {
		t.Fatalf("GetMySchedule 应成功: %v", err)
	}
	// user-1 应至少分配到排班表中
	if result == nil {
		t.Fatal("结果不应为 nil")
	}
	if result.ID == "" {
		t.Error("排班表ID不应为空")
	}
	if result.SemesterID != "sem-1" {
		t.Errorf("期望 semester_id=sem-1，实际=%s", result.SemesterID)
	}
}

// ════════════════════════════════════════════════════════════
// Publish 测试
// ════════════════════════════════════════════════════════════

func TestScheduleService_Publish_Success(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 先执行自动排班
	autoReq := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	autoResult, _ := svc.AutoSchedule(context.Background(), autoReq, "admin-1")

	// 发布
	pubReq := &dto.PublishScheduleRequest{ScheduleID: autoResult.Schedule.ID}
	result, err := svc.Publish(context.Background(), pubReq, "admin-1")
	if err != nil {
		t.Fatalf("Publish 应成功: %v", err)
	}
	if result.Status != "published" {
		t.Errorf("期望 status=published，实际=%s", result.Status)
	}
	if result.PublishedAt == nil {
		t.Error("PublishedAt 不应为 nil")
	}
}

func TestScheduleService_Publish_NotDraft(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 创建一个已发布的排班
	repos.schedule.schedules["pub-sched"] = &model.Schedule{
		ScheduleID: "pub-sched",
		SemesterID: "sem-1",
		Status:     "archived",
	}

	pubReq := &dto.PublishScheduleRequest{ScheduleID: "pub-sched"}
	_, err := svc.Publish(context.Background(), pubReq, "admin-1")
	if !errors.Is(err, ErrScheduleCannotPublish) {
		t.Errorf("期望 ErrScheduleCannotPublish，实际: %v", err)
	}
}

// ════════════════════════════════════════════════════════════
// UpdateItem 测试（draft 状态）
// ════════════════════════════════════════════════════════════

func TestScheduleService_UpdateItem_Success(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 先执行自动排班
	autoReq := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	autoResult, _ := svc.AutoSchedule(context.Background(), autoReq, "admin-1")

	if len(autoResult.Schedule.Items) == 0 {
		t.Skip("无排班项，跳过")
	}

	// 修改第一个排班项的人员
	itemID := autoResult.Schedule.Items[0].ID
	newMember := "user-2"
	updateReq := &dto.UpdateScheduleItemRequest{MemberID: &newMember}

	_, err := svc.UpdateItem(context.Background(), itemID, updateReq, "admin-1")
	if err != nil {
		t.Fatalf("UpdateItem 应成功: %v", err)
	}
}

func TestScheduleService_UpdateItem_NotDraft(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 创建一个已发布的排班和排班项
	repos.schedule.schedules["pub-sched"] = &model.Schedule{
		ScheduleID: "pub-sched",
		SemesterID: "sem-1",
		Status:     "published",
	}
	semID := "sem-1"
	repos.scheduleItem.items["item-pub"] = &model.ScheduleItem{
		ScheduleItemID: "item-pub",
		ScheduleID:     "pub-sched",
		WeekNumber:     1,
		TimeSlotID:     "ts-1",
		MemberID:       "user-1",
		TimeSlot: &model.TimeSlot{
			TimeSlotID: "ts-1", Name: "周一上午", SemesterID: &semID,
			DayOfWeek: 1, StartTime: "08:10", EndTime: "10:05",
		},
	}

	newMember := "user-2"
	req := &dto.UpdateScheduleItemRequest{MemberID: &newMember}
	_, err := svc.UpdateItem(context.Background(), "item-pub", req, "admin-1")
	if !errors.Is(err, ErrScheduleNotDraft) {
		t.Errorf("期望 ErrScheduleNotDraft，实际: %v", err)
	}
}

// ════════════════════════════════════════════════════════════
// UpdatePublishedItem 测试
// ════════════════════════════════════════════════════════════

func TestScheduleService_UpdatePublishedItem_Success(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 创建一个已发布的排班
	repos.schedule.schedules["pub-sched"] = &model.Schedule{
		ScheduleID: "pub-sched",
		SemesterID: "sem-1",
		Status:     "published",
	}
	semID := "sem-1"
	repos.scheduleItem.items["item-pub"] = &model.ScheduleItem{
		ScheduleItemID: "item-pub",
		ScheduleID:     "pub-sched",
		WeekNumber:     1,
		TimeSlotID:     "ts-1",
		MemberID:       "user-1",
		TimeSlot: &model.TimeSlot{
			TimeSlotID: "ts-1", Name: "周一上午", SemesterID: &semID,
			DayOfWeek: 1, StartTime: "08:10", EndTime: "10:05",
		},
	}

	req := &dto.UpdatePublishedItemRequest{
		MemberID: "user-2",
		Reason:   "人员调整",
	}
	_, err := svc.UpdatePublishedItem(context.Background(), "item-pub", req, "admin-1")
	if err != nil {
		t.Fatalf("UpdatePublishedItem 应成功: %v", err)
	}

	// 验证变更日志已记录
	if len(repos.changeLog.logs) == 0 {
		t.Error("应有变更日志记录")
	}
	if repos.changeLog.logs[0].OriginalMemberID != "user-1" {
		t.Errorf("原成员应为 user-1，实际=%s", repos.changeLog.logs[0].OriginalMemberID)
	}
	if repos.changeLog.logs[0].NewMemberID != "user-2" {
		t.Errorf("新成员应为 user-2，实际=%s", repos.changeLog.logs[0].NewMemberID)
	}
}

func TestScheduleService_UpdatePublishedItem_NotPublished(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	repos.schedule.schedules["draft-sched"] = &model.Schedule{
		ScheduleID: "draft-sched",
		SemesterID: "sem-1",
		Status:     "draft",
	}
	repos.scheduleItem.items["item-draft"] = &model.ScheduleItem{
		ScheduleItemID: "item-draft",
		ScheduleID:     "draft-sched",
		WeekNumber:     1,
		TimeSlotID:     "ts-1",
		MemberID:       "user-1",
	}

	req := &dto.UpdatePublishedItemRequest{MemberID: "user-2", Reason: "test"}
	_, err := svc.UpdatePublishedItem(context.Background(), "item-draft", req, "admin-1")
	if !errors.Is(err, ErrScheduleNotPublished) {
		t.Errorf("期望 ErrScheduleNotPublished，实际: %v", err)
	}
}

// ════════════════════════════════════════════════════════════
// ListChangeLogs 测试
// ════════════════════════════════════════════════════════════

func TestScheduleService_ListChangeLogs_Success(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 添加日志
	repos.changeLog.logs = append(repos.changeLog.logs, model.ScheduleChangeLog{
		ChangeLogID:      "log-1",
		ScheduleID:       "sched-1",
		ScheduleItemID:   "item-1",
		OriginalMemberID: "user-1",
		NewMemberID:      "user-2",
		ChangeType:       "admin_modify",
		Reason:           "测试",
		OperatorID:       "admin-1",
	})

	req := &dto.ScheduleChangeLogListRequest{ScheduleID: "sched-1"}
	logs, total, err := svc.ListChangeLogs(context.Background(), req)
	if err != nil {
		t.Fatalf("ListChangeLogs 应成功: %v", err)
	}
	if total != 1 {
		t.Errorf("期望 total=1，实际=%d", total)
	}
	if len(logs) != 1 {
		t.Errorf("期望1条日志，实际=%d", len(logs))
	}
}

// ════════════════════════════════════════════════════════════
// CheckScope 测试
// ════════════════════════════════════════════════════════════

func TestScheduleService_CheckScope_NoChange(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 先自动排班（会保存快照）
	autoReq := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	autoResult, _ := svc.AutoSchedule(context.Background(), autoReq, "admin-1")

	result, err := svc.CheckScope(context.Background(), autoResult.Schedule.ID)
	if err != nil {
		t.Fatalf("CheckScope 应成功: %v", err)
	}
	if result.Changed {
		t.Error("期望 Changed=false（候选人未变化）")
	}
}

func TestScheduleService_CheckScope_MemberAdded(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 先自动排班
	autoReq := &dto.AutoScheduleRequest{SemesterID: "sem-1"}
	autoResult, _ := svc.AutoSchedule(context.Background(), autoReq, "admin-1")

	// 发布
	pubReq := &dto.PublishScheduleRequest{ScheduleID: autoResult.Schedule.ID}
	svc.Publish(context.Background(), pubReq, "admin-1")

	// 添加新候选人
	dept3 := &model.Department{DepartmentID: "dept-3", Name: "设计部"}
	user3 := &model.User{UserID: "user-3", Name: "王五", StudentID: "2021003", DepartmentID: "dept-3", Department: dept3}
	repos.assignment.assignments = append(repos.assignment.assignments, model.UserSemesterAssignment{
		AssignmentID: "a-3", UserID: "user-3", SemesterID: "sem-1",
		DutyRequired: true, TimetableStatus: "submitted", User: user3,
	})

	result, err := svc.CheckScope(context.Background(), autoResult.Schedule.ID)
	if err != nil {
		t.Fatalf("CheckScope 应成功: %v", err)
	}
	if !result.Changed {
		t.Error("期望 Changed=true（新增了候选人）")
	}
	if len(result.AddedUsers) != 1 {
		t.Errorf("期望新增1人，实际=%d", len(result.AddedUsers))
	}
}

// ════════════════════════════════════════════════════════════
// 算法辅助函数测试
// ════════════════════════════════════════════════════════════

func TestWeekNumberToType(t *testing.T) {
	tests := []struct {
		weekNumber    int
		firstWeekType string
		expected      string
	}{
		{1, "odd", "odd"},
		{2, "odd", "even"},
		{1, "even", "even"},
		{2, "even", "odd"},
	}

	for _, tt := range tests {
		result := weekNumberToType(tt.weekNumber, tt.firstWeekType)
		if result != tt.expected {
			t.Errorf("weekNumberToType(%d, %s) = %s, 期望 %s", tt.weekNumber, tt.firstWeekType, result, tt.expected)
		}
	}
}

func TestHasTimeConflict(t *testing.T) {
	tests := []struct {
		name           string
		courseDOW      int
		courseStart    string
		courseEnd      string
		courseWeek     string
		slotDOW        int
		slotStart      string
		slotEnd        string
		slotWeek       string
		expectConflict bool
	}{
		{"完全重叠", 1, "08:00", "10:00", "all", 1, "08:00", "10:00", "odd", true},
		{"部分重叠", 1, "09:00", "11:00", "all", 1, "08:00", "10:00", "odd", true},
		{"不同星期", 2, "08:00", "10:00", "all", 1, "08:00", "10:00", "odd", false},
		{"无重叠", 1, "10:00", "12:00", "all", 1, "08:00", "10:00", "odd", false},
		{"周次不匹配", 1, "08:00", "10:00", "odd", 1, "08:00", "10:00", "even", false},
		{"课程all匹配slot单周", 1, "08:00", "10:00", "all", 1, "08:00", "10:00", "odd", true},
	}

	for _, tt := range tests {
		result := hasTimeConflict(tt.courseDOW, tt.courseStart, tt.courseEnd, tt.courseWeek,
			tt.slotDOW, tt.slotStart, tt.slotEnd, tt.slotWeek)
		if result != tt.expectConflict {
			t.Errorf("%s: hasTimeConflict = %v, 期望 %v", tt.name, result, tt.expectConflict)
		}
	}
}

func TestHasUnavailableConflict(t *testing.T) {
	ut := model.UnavailableTime{
		DayOfWeek:  1,
		StartTime:  "13:00",
		EndTime:    "17:00",
		RepeatType: "weekly",
		WeekType:   "all",
	}

	if !hasUnavailableConflict(ut, 1, "14:00", "16:00", "odd") {
		t.Error("应检测到重叠")
	}

	if hasUnavailableConflict(ut, 1, "08:00", "10:00", "odd") {
		t.Error("不应检测到重叠")
	}

	if hasUnavailableConflict(ut, 2, "14:00", "16:00", "odd") {
		t.Error("不同星期不应冲突")
	}
}

// ════════════════════════════════════════════════════════════
// ValidateCandidate 测试
// ════════════════════════════════════════════════════════════

func TestScheduleService_ValidateCandidate_Available(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 创建排班和排班项
	repos.schedule.schedules["sched-1"] = &model.Schedule{
		ScheduleID: "sched-1",
		SemesterID: "sem-1",
		Status:     "draft",
	}
	semID := "sem-1"
	repos.scheduleItem.items["item-1"] = &model.ScheduleItem{
		ScheduleItemID: "item-1",
		ScheduleID:     "sched-1",
		WeekNumber:     1,
		TimeSlotID:     "ts-2",
		MemberID:       "user-1",
		TimeSlot: &model.TimeSlot{
			TimeSlotID: "ts-2", Name: "周一下午", SemesterID: &semID,
			DayOfWeek: 1, StartTime: "14:00", EndTime: "16:00",
		},
	}

	req := &dto.ValidateCandidateRequest{MemberID: "user-2"}
	result, err := svc.ValidateCandidate(context.Background(), "item-1", req)
	if err != nil {
		t.Fatalf("ValidateCandidate 应成功: %v", err)
	}
	if !result.Valid {
		t.Errorf("应可用，但有冲突: %v", result.Conflicts)
	}
}

func TestScheduleService_ValidateCandidate_WithConflict(t *testing.T) {
	svc, repos := setupTestScheduleService()
	seedBasicData(repos)

	// 给 user-2 添加课程冲突
	repos.courseSchedule.courses = []model.CourseSchedule{
		{
			CourseScheduleID: "cs-1", UserID: "user-2", SemesterID: "sem-1",
			CourseName: "数据结构", DayOfWeek: 1,
			StartTime: "14:00", EndTime: "16:00", WeekType: "all",
		},
	}

	repos.schedule.schedules["sched-1"] = &model.Schedule{
		ScheduleID: "sched-1",
		SemesterID: "sem-1",
		Status:     "draft",
	}
	semID := "sem-1"
	repos.scheduleItem.items["item-1"] = &model.ScheduleItem{
		ScheduleItemID: "item-1",
		ScheduleID:     "sched-1",
		WeekNumber:     1,
		TimeSlotID:     "ts-2",
		MemberID:       "user-1",
		TimeSlot: &model.TimeSlot{
			TimeSlotID: "ts-2", Name: "周一下午", SemesterID: &semID,
			DayOfWeek: 1, StartTime: "14:00", EndTime: "16:00",
		},
	}

	req := &dto.ValidateCandidateRequest{MemberID: "user-2"}
	result, err := svc.ValidateCandidate(context.Background(), "item-1", req)
	if err != nil {
		t.Fatalf("ValidateCandidate 应成功: %v", err)
	}
	if result.Valid {
		t.Error("应不可用（有课程冲突）")
	}
	if len(result.Conflicts) == 0 {
		t.Error("应有冲突原因")
	}
}
