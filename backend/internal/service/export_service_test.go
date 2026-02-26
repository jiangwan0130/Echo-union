package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ── 测试辅助 ──

func setupTestExportService() (ExportService, *mockScheduleRepo, *mockScheduleItemRepo, *mockSemesterRepo) {
	schedRepo := newMockScheduleRepo()
	itemRepo := newMockScheduleItemRepo()
	semRepo := newMockSemesterRepo()
	repo := &repository.Repository{
		User:                   newMockUserRepo(),
		Department:             newMockDeptRepo(),
		Semester:               semRepo,
		TimeSlot:               newMockTimeSlotRepo(),
		Location:               newMockLocationRepo(),
		SystemConfig:           newMockSystemConfigRepo(),
		ScheduleRule:           newMockScheduleRuleRepo(),
		UserSemesterAssignment: newMockUserSemesterAssignmentRepo(),
		Schedule:               schedRepo,
		ScheduleItem:           itemRepo,
		ScheduleMemberSnapshot: newMockScheduleMemberSnapshotRepo(),
		ScheduleChangeLog:      newMockScheduleChangeLogRepo(),
	}
	logger := zap.NewNop()
	svc := NewExportService(repo, logger)
	return svc, schedRepo, itemRepo, semRepo
}

// ── ExportSchedule 测试 ──

func TestExportService_ExportSchedule_NoSchedule(t *testing.T) {
	svc, _, _, _ := setupTestExportService()

	_, _, err := svc.ExportSchedule(context.Background(), "nonexistent-sem")
	if !errors.Is(err, ErrExportNoSchedule) {
		t.Errorf("期望 ErrExportNoSchedule，实际: %v", err)
	}
}

func TestExportService_ExportSchedule_NoItems(t *testing.T) {
	svc, schedRepo, _, _ := setupTestExportService()

	// 创建排班表但不创建排班项
	_ = schedRepo.Create(context.Background(), &model.Schedule{
		SemesterID: "sem-1",
		Status:     "published",
	})

	_, _, err := svc.ExportSchedule(context.Background(), "sem-1")
	if !errors.Is(err, ErrExportNoItems) {
		t.Errorf("期望 ErrExportNoItems，实际: %v", err)
	}
}

func TestExportService_ExportSchedule_Success(t *testing.T) {
	svc, schedRepo, itemRepo, _ := setupTestExportService()

	// 创建排班表
	schedule := &model.Schedule{
		SemesterID: "sem-1",
		Status:     "published",
		Semester:   &model.Semester{SemesterID: "sem-1", Name: "2024秋季学期"},
	}
	_ = schedRepo.Create(context.Background(), schedule)

	// 创建排班项（含关联的时间段和成员信息）
	dept := &model.Department{DepartmentID: "dept-1", Name: "技术部"}
	member := &model.User{UserID: "user-1", Name: "张三", Department: dept}
	ts := &model.TimeSlot{
		TimeSlotID: "ts-1",
		Name:       "上午班",
		DayOfWeek:  1,
		StartTime:  "09:00",
		EndTime:    "12:00",
		IsActive:   true,
	}

	items := []model.ScheduleItem{
		{
			ScheduleID: schedule.ScheduleID,
			WeekNumber: 1,
			TimeSlotID: ts.TimeSlotID,
			MemberID:   member.UserID,
			TimeSlot:   ts,
			Member:     member,
		},
		{
			ScheduleID: schedule.ScheduleID,
			WeekNumber: 2,
			TimeSlotID: ts.TimeSlotID,
			MemberID:   member.UserID,
			TimeSlot:   ts,
			Member:     member,
		},
	}
	_ = itemRepo.BatchCreate(context.Background(), items)

	buf, filename, err := svc.ExportSchedule(context.Background(), "sem-1")
	if err != nil {
		t.Fatalf("ExportSchedule 应成功: %v", err)
	}
	if buf == nil || buf.Len() == 0 {
		t.Error("导出的 Excel buffer 不应为空")
	}
	if filename == "" {
		t.Error("文件名不应为空")
	}
	// Excel .xlsx 文件以 PK (0x504B) 开头
	if buf.Len() > 2 {
		header := buf.Bytes()[:2]
		if header[0] != 0x50 || header[1] != 0x4B {
			t.Error("输出内容不是有效的 xlsx 文件格式（应以 PK 开头）")
		}
	}
}

func TestExportService_ExportSchedule_MultipleWeeksAndSlots(t *testing.T) {
	svc, schedRepo, itemRepo, _ := setupTestExportService()

	schedule := &model.Schedule{
		SemesterID: "sem-1",
		Status:     "published",
	}
	_ = schedRepo.Create(context.Background(), schedule)

	dept := &model.Department{DepartmentID: "dept-1", Name: "宣传部"}
	member1 := &model.User{UserID: "user-1", Name: "张三", Department: dept}
	member2 := &model.User{UserID: "user-2", Name: "李四", Department: dept}

	tsMorning := &model.TimeSlot{
		TimeSlotID: "ts-morning",
		Name:       "上午班",
		DayOfWeek:  1,
		StartTime:  "09:00",
		EndTime:    "12:00",
		IsActive:   true,
	}
	tsAfternoon := &model.TimeSlot{
		TimeSlotID: "ts-afternoon",
		Name:       "下午班",
		DayOfWeek:  1,
		StartTime:  "14:00",
		EndTime:    "17:00",
		IsActive:   true,
	}

	items := []model.ScheduleItem{
		{ScheduleID: schedule.ScheduleID, WeekNumber: 1, TimeSlotID: tsMorning.TimeSlotID, MemberID: "user-1", TimeSlot: tsMorning, Member: member1},
		{ScheduleID: schedule.ScheduleID, WeekNumber: 1, TimeSlotID: tsAfternoon.TimeSlotID, MemberID: "user-2", TimeSlot: tsAfternoon, Member: member2},
		{ScheduleID: schedule.ScheduleID, WeekNumber: 2, TimeSlotID: tsMorning.TimeSlotID, MemberID: "user-2", TimeSlot: tsMorning, Member: member2},
		{ScheduleID: schedule.ScheduleID, WeekNumber: 2, TimeSlotID: tsAfternoon.TimeSlotID, MemberID: "user-1", TimeSlot: tsAfternoon, Member: member1},
	}
	_ = itemRepo.BatchCreate(context.Background(), items)

	buf, filename, err := svc.ExportSchedule(context.Background(), "sem-1")
	if err != nil {
		t.Fatalf("ExportSchedule 应成功: %v", err)
	}
	if buf == nil || buf.Len() == 0 {
		t.Error("导出的 Excel buffer 不应为空")
	}
	if filename == "" {
		t.Error("文件名不应为空")
	}
}
