//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	pkgerrors "echo-union/backend/pkg/errors"

	"echo-union/backend/internal/model"
	"echo-union/backend/internal/repository"
)

// ═══════════════════════════════════════════════════════════
// Test Setup
// ═══════════════════════════════════════════════════════════

var testDB *gorm.DB

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=echo_union password=echo_union_password dbname=echo_union_test sslmode=disable TimeZone=Asia/Shanghai"
	}

	var err error
	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法连接测试数据库: %v\n", err)
		os.Exit(1)
	}

	// 自动迁移测试表结构
	err = testDB.AutoMigrate(
		&model.Department{},
		&model.User{},
		&model.Semester{},
		&model.TimeSlot{},
		&model.Location{},
		&model.Schedule{},
		&model.ScheduleItem{},
		&model.ScheduleMemberSnapshot{},
		&model.ScheduleChangeLog{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AutoMigrate 失败: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}

// setupTestData 创建基础测试数据并返回清理函数
func setupTestData(t *testing.T) (dept *model.Department, user *model.User, semester *model.Semester, ts *model.TimeSlot, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	dept = &model.Department{
		Name:     fmt.Sprintf("测试部门-%d", time.Now().UnixNano()),
		IsActive: true,
	}
	if err := testDB.WithContext(ctx).Create(dept).Error; err != nil {
		t.Fatalf("创建部门失败: %v", err)
	}

	user = &model.User{
		Name:         "测试用户",
		StudentID:    fmt.Sprintf("SID%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("test%d@edu.cn", time.Now().UnixNano()),
		PasswordHash: "$2a$10$placeholder",
		Role:         "member",
		DepartmentID: dept.DepartmentID,
	}
	if err := testDB.WithContext(ctx).Create(user).Error; err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	semester = &model.Semester{
		Name:          fmt.Sprintf("测试学期-%d", time.Now().UnixNano()),
		StartDate:     time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		FirstWeekType: "odd",
		IsActive:      true,
		Status:        "active",
	}
	if err := testDB.WithContext(ctx).Create(semester).Error; err != nil {
		t.Fatalf("创建学期失败: %v", err)
	}

	ts = &model.TimeSlot{
		Name:      "上午班",
		StartTime: "08:10",
		EndTime:   "11:45",
		DayOfWeek: 1,
		IsActive:  true,
	}
	if err := testDB.WithContext(ctx).Create(ts).Error; err != nil {
		t.Fatalf("创建时间段失败: %v", err)
	}

	cleanup = func() {
		testDB.Unscoped().Where("time_slot_id = ?", ts.TimeSlotID).Delete(&model.TimeSlot{})
		testDB.Unscoped().Where("semester_id = ?", semester.SemesterID).Delete(&model.Semester{})
		testDB.Unscoped().Where("user_id = ?", user.UserID).Delete(&model.User{})
		testDB.Unscoped().Where("department_id = ?", dept.DepartmentID).Delete(&model.Department{})
	}
	return
}

// ═══════════════════════════════════════════════════════════
// Test: Transaction Rollback
// ═══════════════════════════════════════════════════════════

func TestTransaction_Rollback(t *testing.T) {
	dept, _, semester, _, cleanup := setupTestData(t)
	defer cleanup()
	_ = zap.NewNop()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	// 开启事务
	tx, err := repo.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx 失败: %v", err)
	}

	txRepo := repo.WithTx(tx)

	// 在事务内创建 Schedule
	sched := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	if err := txRepo.Schedule.Create(ctx, sched); err != nil {
		tx.Rollback()
		t.Fatalf("事务内创建 Schedule 失败: %v", err)
	}

	// 回滚事务
	tx.Rollback()

	// 验证数据未持久化
	_, err = repo.Schedule.GetByID(ctx, sched.ScheduleID)
	if err == nil {
		// 手动清理
		testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).Delete(&model.Schedule{})
		t.Fatal("期望回滚后查不到 Schedule，但实际查到了")
	}
	_ = dept
}

func TestTransaction_Commit(t *testing.T) {
	_, _, semester, _, cleanup := setupTestData(t)
	defer cleanup()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	tx, err := repo.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx 失败: %v", err)
	}

	txRepo := repo.WithTx(tx)

	sched := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	if err := txRepo.Schedule.Create(ctx, sched); err != nil {
		tx.Rollback()
		t.Fatalf("事务内创建 Schedule 失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		t.Fatalf("Commit 失败: %v", err)
	}

	// 验证数据已持久化
	found, err := repo.Schedule.GetByID(ctx, sched.ScheduleID)
	if err != nil {
		t.Fatalf("提交后查询 Schedule 失败: %v", err)
	}
	if found.ScheduleID != sched.ScheduleID {
		t.Errorf("ID 不匹配: expected %s, got %s", sched.ScheduleID, found.ScheduleID)
	}

	// 清理
	testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).Delete(&model.Schedule{})
}

// ═══════════════════════════════════════════════════════════
// Test: Optimistic Lock
// ═══════════════════════════════════════════════════════════

func TestOptimisticLock_Schedule_ConflictDetected(t *testing.T) {
	_, _, semester, _, cleanup := setupTestData(t)
	defer cleanup()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	// 创建 Schedule
	sched := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	if err := repo.Schedule.Create(ctx, sched); err != nil {
		t.Fatalf("创建 Schedule 失败: %v", err)
	}
	defer testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).Delete(&model.Schedule{})

	// 模拟并发：获取两份副本
	copy1, _ := repo.Schedule.GetByID(ctx, sched.ScheduleID)
	copy2, _ := repo.Schedule.GetByID(ctx, sched.ScheduleID)

	// 第一次更新成功
	copy1.Status = "published"
	now := time.Now()
	copy1.PublishedAt = &now
	if err := repo.Schedule.Update(ctx, copy1); err != nil {
		t.Fatalf("第一次更新应成功: %v", err)
	}

	// 第二次更新应失败（version 已过期）
	copy2.Status = "need_regen"
	err := repo.Schedule.Update(ctx, copy2)
	if err == nil {
		t.Fatal("期望乐观锁冲突错误，但更新成功了")
	}
	if err != pkgerrors.ErrOptimisticLock {
		t.Errorf("期望 ErrOptimisticLock，得到: %v", err)
	}
}

func TestOptimisticLock_ScheduleItem_ConflictDetected(t *testing.T) {
	_, user, semester, ts, cleanup := setupTestData(t)
	defer cleanup()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	sched := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	if err := repo.Schedule.Create(ctx, sched); err != nil {
		t.Fatalf("创建 Schedule 失败: %v", err)
	}
	defer testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).Delete(&model.Schedule{})

	items := []model.ScheduleItem{{
		ScheduleID: sched.ScheduleID,
		WeekNumber: 1,
		TimeSlotID: ts.TimeSlotID,
		MemberID:   user.UserID,
	}}
	if err := repo.ScheduleItem.BatchCreate(ctx, items); err != nil {
		t.Fatalf("创建 ScheduleItem 失败: %v", err)
	}
	itemID := items[0].ScheduleItemID
	defer testDB.Unscoped().Where("schedule_item_id = ?", itemID).Delete(&model.ScheduleItem{})

	// 获取两份副本
	copy1, _ := repo.ScheduleItem.GetByID(ctx, itemID)
	copy2, _ := repo.ScheduleItem.GetByID(ctx, itemID)

	// 第一次更新成功
	copy1.WeekNumber = 2
	if err := repo.ScheduleItem.Update(ctx, copy1); err != nil {
		t.Fatalf("第一次更新应成功: %v", err)
	}

	// 第二次更新应失败
	copy2.WeekNumber = 1
	err := repo.ScheduleItem.Update(ctx, copy2)
	if err == nil {
		t.Fatal("期望乐观锁冲突错误，但更新成功了")
	}
	if err != pkgerrors.ErrOptimisticLock {
		t.Errorf("期望 ErrOptimisticLock，得到: %v", err)
	}
}

func TestOptimisticLock_VersionIncrement(t *testing.T) {
	_, _, semester, _, cleanup := setupTestData(t)
	defer cleanup()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	sched := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	if err := repo.Schedule.Create(ctx, sched); err != nil {
		t.Fatalf("创建 Schedule 失败: %v", err)
	}
	defer testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).Delete(&model.Schedule{})

	if sched.Version != 1 {
		t.Errorf("初始 version 应为 1，得到: %d", sched.Version)
	}

	// 连续更新 3 次
	for i := 0; i < 3; i++ {
		got, _ := repo.Schedule.GetByID(ctx, sched.ScheduleID)
		got.Status = "draft"
		if err := repo.Schedule.Update(ctx, got); err != nil {
			t.Fatalf("第 %d 次更新失败: %v", i+1, err)
		}
	}

	// 验证 version 递增到 4
	final, _ := repo.Schedule.GetByID(ctx, sched.ScheduleID)
	if final.Version != 4 {
		t.Errorf("期望 version=4，得到: %d", final.Version)
	}
}

// ═══════════════════════════════════════════════════════════
// Test: Unique Constraint (one active schedule per semester)
// ═══════════════════════════════════════════════════════════

func TestUniqueActiveSchedulePerSemester(t *testing.T) {
	_, _, semester, _, cleanup := setupTestData(t)
	defer cleanup()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	// 创建第一个 draft schedule
	sched1 := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	if err := repo.Schedule.Create(ctx, sched1); err != nil {
		t.Fatalf("创建第一个 Schedule 失败: %v", err)
	}
	defer testDB.Unscoped().Where("schedule_id = ?", sched1.ScheduleID).Delete(&model.Schedule{})

	// 创建第二个 draft schedule（同学期）——应违反唯一约束
	sched2 := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	err := repo.Schedule.Create(ctx, sched2)
	if err == nil {
		// 如果未报错则手动清理并报告失败
		testDB.Unscoped().Where("schedule_id = ?", sched2.ScheduleID).Delete(&model.Schedule{})
		t.Fatal("期望唯一约束违反，但创建成功了。确保已运行 init.sql 中的 uk_schedules_active_per_semester 索引")
	}

	// archived 状态不受唯一约束限制
	sched3 := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "archived",
	}
	if err := repo.Schedule.Create(ctx, sched3); err != nil {
		t.Fatalf("创建 archived Schedule 应成功: %v", err)
	}
	defer testDB.Unscoped().Where("schedule_id = ?", sched3.ScheduleID).Delete(&model.Schedule{})
}

// ═══════════════════════════════════════════════════════════
// Test: Batch Operations
// ═══════════════════════════════════════════════════════════

func TestScheduleItem_BatchCreate(t *testing.T) {
	_, user, semester, ts, cleanup := setupTestData(t)
	defer cleanup()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	sched := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	if err := repo.Schedule.Create(ctx, sched); err != nil {
		t.Fatalf("创建 Schedule 失败: %v", err)
	}
	defer testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).Delete(&model.Schedule{})

	// 批量创建 10 条排班项
	items := make([]model.ScheduleItem, 10)
	for i := range items {
		items[i] = model.ScheduleItem{
			ScheduleID: sched.ScheduleID,
			WeekNumber: (i % 2) + 1,
			TimeSlotID: ts.TimeSlotID,
			MemberID:   user.UserID,
		}
	}

	if err := repo.ScheduleItem.BatchCreate(ctx, items); err != nil {
		t.Fatalf("BatchCreate 失败: %v", err)
	}

	// 验证所有项已创建
	list, err := repo.ScheduleItem.ListBySchedule(ctx, sched.ScheduleID)
	if err != nil {
		t.Fatalf("ListBySchedule 失败: %v", err)
	}
	if len(list) != 10 {
		t.Errorf("期望 10 条排班项，得到 %d 条", len(list))
	}

	// 清理
	testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).Delete(&model.ScheduleItem{})
}

// ═══════════════════════════════════════════════════════════
// Test: User ListByIDs
// ═══════════════════════════════════════════════════════════

func TestUser_ListByIDs(t *testing.T) {
	dept, user, _, _, cleanup := setupTestData(t)
	defer cleanup()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	// 创建第二个用户
	user2 := &model.User{
		Name:         "第二用户",
		StudentID:    fmt.Sprintf("SID2%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("test2%d@edu.cn", time.Now().UnixNano()),
		PasswordHash: "$2a$10$placeholder",
		Role:         "member",
		DepartmentID: dept.DepartmentID,
	}
	if err := testDB.WithContext(ctx).Create(user2).Error; err != nil {
		t.Fatalf("创建第二用户失败: %v", err)
	}
	defer testDB.Unscoped().Where("user_id = ?", user2.UserID).Delete(&model.User{})

	// 批量查询
	users, err := repo.User.ListByIDs(ctx, []string{user.UserID, user2.UserID})
	if err != nil {
		t.Fatalf("ListByIDs 失败: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("期望 2 个用户，得到 %d 个", len(users))
	}

	// 空 ID 列表
	users, err = repo.User.ListByIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("空 ID 列表不应报错: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("空 ID 列表期望返回 0 个用户，得到 %d 个", len(users))
	}
}

// ═══════════════════════════════════════════════════════════
// Test: Soft Delete
// ═══════════════════════════════════════════════════════════

func TestSchedule_SoftDelete(t *testing.T) {
	_, _, semester, _, cleanup := setupTestData(t)
	defer cleanup()

	repo := repository.NewRepository(testDB)
	ctx := context.Background()

	sched := &model.Schedule{
		SemesterID: semester.SemesterID,
		Status:     "draft",
	}
	if err := repo.Schedule.Create(ctx, sched); err != nil {
		t.Fatalf("创建 Schedule 失败: %v", err)
	}
	defer testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).Delete(&model.Schedule{})

	// 软删除
	if err := repo.Schedule.Delete(ctx, sched.ScheduleID); err != nil {
		t.Fatalf("软删除失败: %v", err)
	}

	// 常规查询应找不到
	_, err := repo.Schedule.GetByID(ctx, sched.ScheduleID)
	if err == nil {
		t.Fatal("软删除后应查不到记录")
	}

	// Unscoped 查询应能找到
	var found model.Schedule
	err = testDB.Unscoped().Where("schedule_id = ?", sched.ScheduleID).First(&found).Error
	if err != nil {
		t.Fatalf("Unscoped 查询应能找到: %v", err)
	}
	if found.DeletedAt.Time.IsZero() {
		t.Error("DeletedAt 应已设置")
	}
}
