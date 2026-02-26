package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ════════════════════════════════════════════════════════════
// ICS 解析器测试
// ════════════════════════════════════════════════════════════

// 标准 ICS 测试数据：2 门周重复课程 + 1 门单次事件
const testICSContent = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
SUMMARY:高等数学
DTSTART;TZID=Asia/Shanghai:20250224T081000
DTEND;TZID=Asia/Shanghai:20250224T100500
RRULE:FREQ=WEEKLY;COUNT=16
END:VEVENT
BEGIN:VEVENT
SUMMARY:大学英语
DTSTART;TZID=Asia/Shanghai:20250225T140000
DTEND;TZID=Asia/Shanghai:20250225T160000
RRULE:FREQ=WEEKLY;COUNT=16
END:VEVENT
BEGIN:VEVENT
SUMMARY:专题讲座
DTSTART;TZID=Asia/Shanghai:20250303T090000
DTEND;TZID=Asia/Shanghai:20250303T110000
END:VEVENT
END:VCALENDAR`

// 双周课 ICS
const testICSBiweekly = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
SUMMARY:物理实验
DTSTART;TZID=Asia/Shanghai:20250224T081000
DTEND;TZID=Asia/Shanghai:20250224T100500
RRULE:FREQ=WEEKLY;INTERVAL=2;COUNT=8
END:VEVENT
END:VCALENDAR`

func TestParseICS_BasicCourses(t *testing.T) {
	semStart := time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC)
	semEnd := time.Date(2025, 7, 6, 0, 0, 0, 0, time.UTC)

	reader := strings.NewReader(testICSContent)
	courses, err := ParseICS(reader, "user-1", "sem-1", semStart, semEnd)
	if err != nil {
		t.Fatalf("ParseICS 失败: %v", err)
	}

	if len(courses) != 3 {
		t.Fatalf("期望 3 门课程, 实际 %d 门", len(courses))
	}

	// 校验高等数学（周一, 16 周重复）
	var math *model.CourseSchedule
	for i, c := range courses {
		if c.CourseName == "高等数学" {
			math = &courses[i]
			break
		}
	}
	if math == nil {
		t.Fatal("未找到高等数学")
	}
	if math.DayOfWeek != 1 {
		t.Errorf("高等数学 DayOfWeek 期望 1, 实际 %d", math.DayOfWeek)
	}
	if math.StartTime != "08:10" {
		t.Errorf("高等数学 StartTime 期望 08:10, 实际 %s", math.StartTime)
	}
	if len(math.Weeks) != 16 {
		t.Errorf("高等数学 Weeks 数量期望 16, 实际 %d", len(math.Weeks))
	}
	if math.WeekType != "all" {
		t.Errorf("高等数学 WeekType 期望 all, 实际 %s", math.WeekType)
	}
	if math.Source != "ics" {
		t.Errorf("Source 期望 ics, 实际 %s", math.Source)
	}

	// 校验专题讲座（单次事件, 第 2 周周一）
	var lecture *model.CourseSchedule
	for i, c := range courses {
		if c.CourseName == "专题讲座" {
			lecture = &courses[i]
			break
		}
	}
	if lecture == nil {
		t.Fatal("未找到专题讲座")
	}
	if len(lecture.Weeks) != 1 {
		t.Errorf("专题讲座 Weeks 数量期望 1, 实际 %d", len(lecture.Weeks))
	}
	if lecture.Weeks[0] != 2 {
		t.Errorf("专题讲座 Week 期望 2, 实际 %d", lecture.Weeks[0])
	}
}

func TestParseICS_BiweeklyCourse(t *testing.T) {
	semStart := time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC)
	semEnd := time.Date(2025, 7, 6, 0, 0, 0, 0, time.UTC)

	reader := strings.NewReader(testICSBiweekly)
	courses, err := ParseICS(reader, "user-1", "sem-1", semStart, semEnd)
	if err != nil {
		t.Fatalf("ParseICS 失败: %v", err)
	}

	if len(courses) != 1 {
		t.Fatalf("期望 1 门课程, 实际 %d 门", len(courses))
	}

	phys := courses[0]
	if phys.CourseName != "物理实验" {
		t.Errorf("课程名期望 物理实验, 实际 %s", phys.CourseName)
	}
	// INTERVAL=2, COUNT=8 → 第 1,3,5,7,9,11,13,15 周
	if len(phys.Weeks) != 8 {
		t.Errorf("Weeks 数量期望 8, 实际 %d: %v", len(phys.Weeks), phys.Weeks)
	}
	// 应全为奇数周
	if phys.WeekType != "odd" {
		t.Errorf("WeekType 期望 odd, 实际 %s (weeks: %v)", phys.WeekType, phys.Weeks)
	}
}

func TestParseICS_EmptyCalendar(t *testing.T) {
	icsEmpty := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
END:VCALENDAR`

	semStart := time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC)
	semEnd := time.Date(2025, 7, 6, 0, 0, 0, 0, time.UTC)

	reader := strings.NewReader(icsEmpty)
	courses, err := ParseICS(reader, "user-1", "sem-1", semStart, semEnd)
	if err != nil {
		t.Fatalf("ParseICS 对空日历不应返回 error: %v", err)
	}
	if len(courses) != 0 {
		t.Errorf("空日历期望 0 门课程, 实际 %d", len(courses))
	}
}

func TestParseICS_MergesSameCourse(t *testing.T) {
	// 同一课程以多个单次事件表示
	ics := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
SUMMARY:体育
DTSTART;TZID=Asia/Shanghai:20250224T081000
DTEND;TZID=Asia/Shanghai:20250224T100500
END:VEVENT
BEGIN:VEVENT
SUMMARY:体育
DTSTART;TZID=Asia/Shanghai:20250303T081000
DTEND;TZID=Asia/Shanghai:20250303T100500
END:VEVENT
BEGIN:VEVENT
SUMMARY:体育
DTSTART;TZID=Asia/Shanghai:20250310T081000
DTEND;TZID=Asia/Shanghai:20250310T100500
END:VEVENT
END:VCALENDAR`

	semStart := time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC)
	semEnd := time.Date(2025, 7, 6, 0, 0, 0, 0, time.UTC)

	reader := strings.NewReader(ics)
	courses, err := ParseICS(reader, "user-1", "sem-1", semStart, semEnd)
	if err != nil {
		t.Fatalf("ParseICS 失败: %v", err)
	}

	// 3 个单次事件应合并为 1 门课程
	if len(courses) != 1 {
		t.Fatalf("期望合并为 1 门课程, 实际 %d 门", len(courses))
	}
	if len(courses[0].Weeks) != 3 {
		t.Errorf("合并后 Weeks 期望 3 周, 实际 %d: %v", len(courses[0].Weeks), courses[0].Weeks)
	}
}

// ── 辅助函数测试 ──

func TestDeriveWeekType(t *testing.T) {
	tests := []struct {
		weeks    []int
		expected string
	}{
		{[]int{1, 3, 5, 7}, "odd"},
		{[]int{2, 4, 6, 8}, "even"},
		{[]int{1, 2, 3}, "all"},
		{[]int{}, "all"},
		{nil, "all"},
	}
	for _, tt := range tests {
		result := deriveWeekType(tt.weeks)
		if result != tt.expected {
			t.Errorf("deriveWeekType(%v) = %s, 期望 %s", tt.weeks, result, tt.expected)
		}
	}
}

func TestDateToWeekNumber(t *testing.T) {
	sem := time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		date     time.Time
		expected int
	}{
		{time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC), 1},
		{time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC), 1},
		{time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC), 2},
		{time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC), 3},
	}
	for _, tt := range tests {
		result := dateToWeekNumber(tt.date, sem)
		if result != tt.expected {
			t.Errorf("dateToWeekNumber(%v) = %d, 期望 %d", tt.date, result, tt.expected)
		}
	}
}

func TestGoWeekdayToISO(t *testing.T) {
	tests := []struct {
		wd       time.Weekday
		expected int
	}{
		{time.Monday, 1},
		{time.Tuesday, 2},
		{time.Friday, 5},
		{time.Sunday, 7},
	}
	for _, tt := range tests {
		result := goWeekdayToISO(tt.wd)
		if result != tt.expected {
			t.Errorf("goWeekdayToISO(%v) = %d, 期望 %d", tt.wd, result, tt.expected)
		}
	}
}

// ════════════════════════════════════════════════════════════
// IntArray 测试
// ════════════════════════════════════════════════════════════

func TestIntArray_ScanAndValue(t *testing.T) {
	var a model.IntArray

	// Scan string
	if err := a.Scan("{1,2,3}"); err != nil {
		t.Fatalf("Scan 失败: %v", err)
	}
	if len(a) != 3 || a[0] != 1 || a[1] != 2 || a[2] != 3 {
		t.Errorf("Scan 结果错误: %v", a)
	}

	// Value
	v, err := a.Value()
	if err != nil {
		t.Fatalf("Value 失败: %v", err)
	}
	if v != "{1,2,3}" {
		t.Errorf("Value 结果期望 '{1,2,3}', 实际 '%v'", v)
	}

	// Scan nil
	if err := a.Scan(nil); err != nil {
		t.Fatalf("Scan nil 失败: %v", err)
	}
	if a != nil {
		t.Errorf("Scan nil 后期望 nil, 实际 %v", a)
	}

	// Value nil
	var nilArray model.IntArray
	v, err = nilArray.Value()
	if err != nil {
		t.Fatalf("Value nil 失败: %v", err)
	}
	if v != nil {
		t.Errorf("Value nil 期望 nil, 实际 %v", v)
	}

	// Scan empty
	var empty model.IntArray
	if err := empty.Scan("{}"); err != nil {
		t.Fatalf("Scan empty 失败: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("Scan empty 结果期望 [], 实际 %v", empty)
	}
}

// ════════════════════════════════════════════════════════════
// TimetableService 测试
// ════════════════════════════════════════════════════════════

// 测试辅助：构建 TimetableService

func setupTestTimetableService() (TimetableService, *testTimetableRepos) {
	repos := &testTimetableRepos{
		semester:       newMockSemesterRepo(),
		courseSchedule: newMockCourseScheduleRepo(),
		unavailable:    newMockUnavailableTimeRepo(),
		assignment:     newMockUserSemesterAssignmentRepo(),
		department:     newMockDeptRepo(),
	}
	repoAgg := &repository.Repository{
		User:                   newMockUserRepo(),
		Department:             repos.department,
		Semester:               repos.semester,
		TimeSlot:               newMockTimeSlotRepo(),
		Location:               newMockLocationRepo(),
		SystemConfig:           newMockSystemConfigRepo(),
		ScheduleRule:           newMockScheduleRuleRepo(),
		CourseSchedule:         repos.courseSchedule,
		UnavailableTime:        repos.unavailable,
		UserSemesterAssignment: repos.assignment,
		Schedule:               newMockScheduleRepo(),
		ScheduleItem:           newMockScheduleItemRepo(),
		ScheduleMemberSnapshot: newMockScheduleMemberSnapshotRepo(),
		ScheduleChangeLog:      newMockScheduleChangeLogRepo(),
	}
	logger := zap.NewNop()
	svc := NewTimetableService(repoAgg, logger)
	return svc, repos
}

type testTimetableRepos struct {
	semester       *mockSemesterRepo
	courseSchedule *mockCourseScheduleRepo
	unavailable    *mockUnavailableTimeRepo
	assignment     *mockUserSemesterAssignmentRepo
	department     *mockDeptRepo
}

func seedTimetableBasicData(repos *testTimetableRepos) {
	repos.semester.semesters["sem-1"] = &model.Semester{
		SemesterID:    "sem-1",
		Name:          "2025-2026春",
		IsActive:      true,
		FirstWeekType: "odd",
		StartDate:     time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2025, 7, 6, 0, 0, 0, 0, time.UTC),
	}

	dept1 := &model.Department{DepartmentID: "dept-1", Name: "技术部"}
	user1 := &model.User{UserID: "user-1", Name: "张三", StudentID: "2021001", DepartmentID: "dept-1", Department: dept1}
	repos.department.departments["dept-1"] = dept1

	repos.assignment.assignments = []model.UserSemesterAssignment{
		{
			AssignmentID:    "a-1",
			UserID:          "user-1",
			SemesterID:      "sem-1",
			DutyRequired:    true,
			TimetableStatus: "not_submitted",
			User:            user1,
		},
	}
}

// ── ImportICS 测试 ──

func TestTimetableService_ImportICS_Success(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	reader := strings.NewReader(testICSContent)
	resp, err := svc.ImportICS(ctx, reader, "user-1", "sem-1")
	if err != nil {
		t.Fatalf("ImportICS 失败: %v", err)
	}
	if resp.ImportedCount != 3 {
		t.Errorf("ImportedCount 期望 3, 实际 %d", resp.ImportedCount)
	}
	if len(resp.Events) != 3 {
		t.Errorf("Events 数量期望 3, 实际 %d", len(resp.Events))
	}

	// 验证数据已持久化
	courses, _ := repos.courseSchedule.ListByUserAndSemester(ctx, "user-1", "sem-1")
	if len(courses) != 3 {
		t.Errorf("持久化课程期望 3 条, 实际 %d 条", len(courses))
	}
}

func TestTimetableService_ImportICS_ReplacesOldData(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	// 先导入一次
	reader1 := strings.NewReader(testICSContent)
	_, _ = svc.ImportICS(ctx, reader1, "user-1", "sem-1")

	// 提交
	_, _ = svc.SubmitTimetable(ctx, "user-1", "sem-1")

	// 验证已提交
	assignment, _ := repos.assignment.GetByUserAndSemester(ctx, "user-1", "sem-1")
	if assignment.TimetableStatus != "submitted" {
		t.Fatal("预期状态为 submitted")
	}

	// 再次导入 → 应全量替换并回退状态
	reader2 := strings.NewReader(testICSBiweekly) // 只有 1 门课
	resp, err := svc.ImportICS(ctx, reader2, "user-1", "sem-1")
	if err != nil {
		t.Fatalf("第二次 ImportICS 失败: %v", err)
	}
	if resp.ImportedCount != 1 {
		t.Errorf("第二次 ImportedCount 期望 1, 实际 %d", resp.ImportedCount)
	}

	// 验证旧数据已清除
	courses, _ := repos.courseSchedule.ListByUserAndSemester(ctx, "user-1", "sem-1")
	if len(courses) != 1 {
		t.Errorf("替换后课程期望 1 条, 实际 %d 条", len(courses))
	}

	// 验证提交状态已回退
	assignment, _ = repos.assignment.GetByUserAndSemester(ctx, "user-1", "sem-1")
	if assignment.TimetableStatus != "not_submitted" {
		t.Errorf("提交状态期望 not_submitted, 实际 %s", assignment.TimetableStatus)
	}
}

func TestTimetableService_ImportICS_NoActiveSemester(t *testing.T) {
	svc, _ := setupTestTimetableService()
	ctx := context.Background()

	reader := strings.NewReader(testICSContent)
	_, err := svc.ImportICS(ctx, reader, "user-1", "")
	if err != ErrTimetableNoActiveSemester {
		t.Errorf("无活动学期期望 ErrTimetableNoActiveSemester, 实际 %v", err)
	}
}

func TestTimetableService_ImportICS_EmptyICS(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	reader := strings.NewReader(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
END:VCALENDAR`)
	_, err := svc.ImportICS(ctx, reader, "user-1", "sem-1")
	if err != ErrTimetableICSEmpty {
		t.Errorf("空 ICS 期望 ErrTimetableICSEmpty, 实际 %v", err)
	}
}

// ── GetMyTimetable 测试 ──

func TestTimetableService_GetMyTimetable(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	// 先导入课表
	reader := strings.NewReader(testICSContent)
	_, _ = svc.ImportICS(ctx, reader, "user-1", "sem-1")

	resp, err := svc.GetMyTimetable(ctx, "user-1", "sem-1")
	if err != nil {
		t.Fatalf("GetMyTimetable 失败: %v", err)
	}
	if len(resp.Courses) != 3 {
		t.Errorf("课程数量期望 3, 实际 %d", len(resp.Courses))
	}
	if resp.SubmitStatus != "not_submitted" {
		t.Errorf("SubmitStatus 期望 not_submitted, 实际 %s", resp.SubmitStatus)
	}
}

// ── 不可用时间 CRUD 测试 ──

func TestTimetableService_UnavailableTime_CRUD(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	// Create
	createResp, err := svc.CreateUnavailableTime(ctx, &CreateUnavailableTimeParams{
		DayOfWeek:  3,
		StartTime:  "14:00",
		EndTime:    "16:00",
		Reason:     "社团活动",
		RepeatType: "weekly",
		WeekType:   "all",
		SemesterID: "sem-1",
	}, "user-1")
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if createResp.DayOfWeek != 3 {
		t.Errorf("DayOfWeek 期望 3, 实际 %d", createResp.DayOfWeek)
	}

	utID := createResp.ID

	// Update
	newReason := "换到射击队训练"
	updateResp, err := svc.UpdateUnavailableTime(ctx, utID, &UpdateUnavailableTimeParams{
		Reason: &newReason,
	}, "user-1")
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updateResp.Reason != newReason {
		t.Errorf("Reason 期望 %s, 实际 %s", newReason, updateResp.Reason)
	}

	// 非本人更新 → 拒绝
	_, err = svc.UpdateUnavailableTime(ctx, utID, &UpdateUnavailableTimeParams{
		Reason: &newReason,
	}, "user-2")
	if err != ErrTimetableUnavailableNotOwner {
		t.Errorf("非本人更新期望 ErrTimetableUnavailableNotOwner, 实际 %v", err)
	}

	// Delete
	err = svc.DeleteUnavailableTime(ctx, utID, "user-1")
	if err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}

	// 确认已删除
	times, _ := repos.unavailable.ListByUserAndSemester(ctx, "user-1", "sem-1")
	if len(times) != 0 {
		t.Errorf("删除后期望 0 条, 实际 %d 条", len(times))
	}
}

// ── Submit 测试 ──

func TestTimetableService_Submit_Success(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	// 先导入课表
	reader := strings.NewReader(testICSContent)
	_, _ = svc.ImportICS(ctx, reader, "user-1", "sem-1")

	// 提交
	resp, err := svc.SubmitTimetable(ctx, "user-1", "sem-1")
	if err != nil {
		t.Fatalf("Submit 失败: %v", err)
	}
	if resp.SubmitStatus != "submitted" {
		t.Errorf("SubmitStatus 期望 submitted, 实际 %s", resp.SubmitStatus)
	}
	if resp.SubmittedAt == nil {
		t.Error("SubmittedAt 不应为 nil")
	}
}

func TestTimetableService_Submit_NoCourses(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	_, err := svc.SubmitTimetable(ctx, "user-1", "sem-1")
	if err != ErrTimetableNoCourses {
		t.Errorf("无课表期望 ErrTimetableNoCourses, 实际 %v", err)
	}
}

// ── Progress 测试 ──

func TestTimetableService_GetProgress(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	// 添加第二个用户
	dept2 := &model.Department{DepartmentID: "dept-2", Name: "运营部"}
	user2 := &model.User{UserID: "user-2", Name: "李四", StudentID: "2021002", DepartmentID: "dept-2", Department: dept2}
	repos.department.departments["dept-2"] = dept2
	repos.assignment.assignments = append(repos.assignment.assignments, model.UserSemesterAssignment{
		AssignmentID:    "a-2",
		UserID:          "user-2",
		SemesterID:      "sem-1",
		DutyRequired:    true,
		TimetableStatus: "submitted",
		User:            user2,
	})

	resp, err := svc.GetProgress(ctx, "sem-1")
	if err != nil {
		t.Fatalf("GetProgress 失败: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("Total 期望 2, 实际 %d", resp.Total)
	}
	if resp.Submitted != 1 {
		t.Errorf("Submitted 期望 1, 实际 %d", resp.Submitted)
	}
	if resp.Progress != 50 {
		t.Errorf("Progress 期望 50, 实际 %f", resp.Progress)
	}
	if len(resp.Departments) != 2 {
		t.Errorf("Department 数量期望 2, 实际 %d", len(resp.Departments))
	}
}

func TestTimetableService_GetDepartmentProgress(t *testing.T) {
	svc, repos := setupTestTimetableService()
	seedTimetableBasicData(repos)
	ctx := context.Background()

	resp, err := svc.GetDepartmentProgress(ctx, "dept-1", "sem-1")
	if err != nil {
		t.Fatalf("GetDepartmentProgress 失败: %v", err)
	}
	if resp.DepartmentName != "技术部" {
		t.Errorf("DepartmentName 期望 技术部, 实际 %s", resp.DepartmentName)
	}
	if resp.Total != 1 {
		t.Errorf("Total 期望 1, 实际 %d", resp.Total)
	}
	if len(resp.Members) != 1 {
		t.Errorf("Members 数量期望 1, 实际 %d", len(resp.Members))
	}
}

// ── 辅助类型（避免在测试中引入 dto 包的类型别名） ──

type CreateUnavailableTimeParams = dto.CreateUnavailableTimeRequest
type UpdateUnavailableTimeParams = dto.UpdateUnavailableTimeRequest
