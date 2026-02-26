package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"echo-union/backend/internal/model"
)

// ── Mock SemesterRepository ──

type mockSemesterRepo struct {
	semesters map[string]*model.Semester
}

func newMockSemesterRepo() *mockSemesterRepo {
	return &mockSemesterRepo{semesters: make(map[string]*model.Semester)}
}

func (m *mockSemesterRepo) Create(_ context.Context, semester *model.Semester) error {
	if semester.SemesterID == "" {
		semester.SemesterID = "sem-" + semester.Name
	}
	m.semesters[semester.SemesterID] = semester
	return nil
}

func (m *mockSemesterRepo) GetByID(_ context.Context, id string) (*model.Semester, error) {
	if s, ok := m.semesters[id]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockSemesterRepo) GetCurrent(_ context.Context) (*model.Semester, error) {
	for _, s := range m.semesters {
		if s.IsActive {
			return s, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockSemesterRepo) List(_ context.Context) ([]model.Semester, error) {
	var result []model.Semester
	for _, s := range m.semesters {
		result = append(result, *s)
	}
	return result, nil
}

func (m *mockSemesterRepo) Update(_ context.Context, semester *model.Semester) error {
	m.semesters[semester.SemesterID] = semester
	return nil
}

func (m *mockSemesterRepo) Delete(_ context.Context, id string, _ string) error {
	delete(m.semesters, id)
	return nil
}

func (m *mockSemesterRepo) ClearActive(_ context.Context) error {
	for _, s := range m.semesters {
		s.IsActive = false
	}
	return nil
}

// ── Mock TimeSlotRepository ──

type mockTimeSlotRepo struct {
	slots map[string]*model.TimeSlot
}

func newMockTimeSlotRepo() *mockTimeSlotRepo {
	return &mockTimeSlotRepo{slots: make(map[string]*model.TimeSlot)}
}

func (m *mockTimeSlotRepo) Create(_ context.Context, slot *model.TimeSlot) error {
	if slot.TimeSlotID == "" {
		slot.TimeSlotID = "ts-" + slot.Name
	}
	m.slots[slot.TimeSlotID] = slot
	return nil
}

func (m *mockTimeSlotRepo) GetByID(_ context.Context, id string) (*model.TimeSlot, error) {
	if s, ok := m.slots[id]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockTimeSlotRepo) List(_ context.Context, semesterID string, dayOfWeek *int) ([]model.TimeSlot, error) {
	var result []model.TimeSlot
	for _, s := range m.slots {
		if !s.IsActive {
			continue
		}
		if semesterID != "" && s.SemesterID != nil && *s.SemesterID != semesterID {
			continue
		}
		if dayOfWeek != nil && s.DayOfWeek != *dayOfWeek {
			continue
		}
		result = append(result, *s)
	}
	return result, nil
}

func (m *mockTimeSlotRepo) Update(_ context.Context, slot *model.TimeSlot) error {
	m.slots[slot.TimeSlotID] = slot
	return nil
}

func (m *mockTimeSlotRepo) Delete(_ context.Context, id string, _ string) error {
	delete(m.slots, id)
	return nil
}

// ── Mock LocationRepository ──

type mockLocationRepo struct {
	locations map[string]*model.Location
}

func newMockLocationRepo() *mockLocationRepo {
	return &mockLocationRepo{locations: make(map[string]*model.Location)}
}

func (m *mockLocationRepo) Create(_ context.Context, loc *model.Location) error {
	if loc.LocationID == "" {
		loc.LocationID = "loc-" + loc.Name
	}
	m.locations[loc.LocationID] = loc
	return nil
}

func (m *mockLocationRepo) GetByID(_ context.Context, id string) (*model.Location, error) {
	if l, ok := m.locations[id]; ok {
		return l, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockLocationRepo) List(_ context.Context, includeInactive bool) ([]model.Location, error) {
	var result []model.Location
	for _, l := range m.locations {
		if !includeInactive && !l.IsActive {
			continue
		}
		result = append(result, *l)
	}
	return result, nil
}

func (m *mockLocationRepo) Update(_ context.Context, loc *model.Location) error {
	m.locations[loc.LocationID] = loc
	return nil
}

func (m *mockLocationRepo) Delete(_ context.Context, id string, _ string) error {
	delete(m.locations, id)
	return nil
}

// ── Mock SystemConfigRepository ──

type mockSystemConfigRepo struct {
	cfg *model.SystemConfig
}

func newMockSystemConfigRepo() *mockSystemConfigRepo {
	return &mockSystemConfigRepo{
		cfg: &model.SystemConfig{
			Singleton:            true,
			SwapDeadlineHours:    24,
			DutyReminderTime:     "09:00",
			DefaultLocation:      "学生会办公室",
			SignInWindowMinutes:  15,
			SignOutWindowMinutes: 15,
		},
	}
}

func (m *mockSystemConfigRepo) Get(_ context.Context) (*model.SystemConfig, error) {
	if m.cfg == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return m.cfg, nil
}

func (m *mockSystemConfigRepo) Update(_ context.Context, cfg *model.SystemConfig) error {
	m.cfg = cfg
	return nil
}

// ── Mock ScheduleRuleRepository ──

type mockScheduleRuleRepo struct {
	rules map[string]*model.ScheduleRule
}

func newMockScheduleRuleRepo() *mockScheduleRuleRepo {
	return &mockScheduleRuleRepo{rules: make(map[string]*model.ScheduleRule)}
}

func (m *mockScheduleRuleRepo) GetByID(_ context.Context, id string) (*model.ScheduleRule, error) {
	if r, ok := m.rules[id]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockScheduleRuleRepo) List(_ context.Context) ([]model.ScheduleRule, error) {
	var result []model.ScheduleRule
	for _, r := range m.rules {
		result = append(result, *r)
	}
	return result, nil
}

func (m *mockScheduleRuleRepo) Update(_ context.Context, rule *model.ScheduleRule) error {
	m.rules[rule.RuleID] = rule
	return nil
}

// ── Mock CourseScheduleRepository ──

type mockCourseScheduleRepo struct {
	courses []model.CourseSchedule
}

func newMockCourseScheduleRepo() *mockCourseScheduleRepo {
	return &mockCourseScheduleRepo{}
}

func (m *mockCourseScheduleRepo) ListByUserAndSemester(_ context.Context, userID, semesterID string) ([]model.CourseSchedule, error) {
	var result []model.CourseSchedule
	for _, c := range m.courses {
		if c.UserID == userID && c.SemesterID == semesterID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockCourseScheduleRepo) ListBySemester(_ context.Context, semesterID string) ([]model.CourseSchedule, error) {
	var result []model.CourseSchedule
	for _, c := range m.courses {
		if c.SemesterID == semesterID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockCourseScheduleRepo) BatchCreate(_ context.Context, courses []model.CourseSchedule) error {
	for i := range courses {
		if courses[i].CourseScheduleID == "" {
			courses[i].CourseScheduleID = fmt.Sprintf("cs-%d", len(m.courses)+i+1)
		}
	}
	m.courses = append(m.courses, courses...)
	return nil
}

func (m *mockCourseScheduleRepo) DeleteByUserAndSemester(_ context.Context, userID, semesterID string) error {
	var remaining []model.CourseSchedule
	for _, c := range m.courses {
		if !(c.UserID == userID && c.SemesterID == semesterID) {
			remaining = append(remaining, c)
		}
	}
	m.courses = remaining
	return nil
}

func (m *mockCourseScheduleRepo) ReplaceByUserAndSemester(_ context.Context, userID, semesterID string, courses []model.CourseSchedule) error {
	_ = m.DeleteByUserAndSemester(context.Background(), userID, semesterID)
	return m.BatchCreate(context.Background(), courses)
}

// ── Mock UnavailableTimeRepository ──

type mockUnavailableTimeRepo struct {
	times     []model.UnavailableTime
	idCounter int
}

func newMockUnavailableTimeRepo() *mockUnavailableTimeRepo {
	return &mockUnavailableTimeRepo{}
}

func (m *mockUnavailableTimeRepo) ListByUserAndSemester(_ context.Context, userID, semesterID string) ([]model.UnavailableTime, error) {
	var result []model.UnavailableTime
	for _, t := range m.times {
		if t.UserID == userID && t.SemesterID == semesterID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockUnavailableTimeRepo) ListBySemester(_ context.Context, semesterID string) ([]model.UnavailableTime, error) {
	var result []model.UnavailableTime
	for _, t := range m.times {
		if t.SemesterID == semesterID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockUnavailableTimeRepo) GetByID(_ context.Context, id string) (*model.UnavailableTime, error) {
	for i, t := range m.times {
		if t.UnavailableTimeID == id {
			return &m.times[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUnavailableTimeRepo) Create(_ context.Context, ut *model.UnavailableTime) error {
	m.idCounter++
	if ut.UnavailableTimeID == "" {
		ut.UnavailableTimeID = fmt.Sprintf("ut-%d", m.idCounter)
	}
	m.times = append(m.times, *ut)
	return nil
}

func (m *mockUnavailableTimeRepo) Update(_ context.Context, ut *model.UnavailableTime) error {
	for i, t := range m.times {
		if t.UnavailableTimeID == ut.UnavailableTimeID {
			m.times[i] = *ut
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockUnavailableTimeRepo) Delete(_ context.Context, id string, _ string) error {
	for i, t := range m.times {
		if t.UnavailableTimeID == id {
			m.times = append(m.times[:i], m.times[i+1:]...)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

// ── Mock UserSemesterAssignmentRepository ──

type mockUserSemesterAssignmentRepo struct {
	assignments []model.UserSemesterAssignment
}

func newMockUserSemesterAssignmentRepo() *mockUserSemesterAssignmentRepo {
	return &mockUserSemesterAssignmentRepo{}
}

func (m *mockUserSemesterAssignmentRepo) ListBySemester(_ context.Context, semesterID string) ([]model.UserSemesterAssignment, error) {
	var result []model.UserSemesterAssignment
	for _, a := range m.assignments {
		if a.SemesterID == semesterID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockUserSemesterAssignmentRepo) ListDutyRequiredSubmitted(_ context.Context, semesterID string) ([]model.UserSemesterAssignment, error) {
	var result []model.UserSemesterAssignment
	for _, a := range m.assignments {
		if a.SemesterID == semesterID && a.DutyRequired && a.TimetableStatus == "submitted" {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockUserSemesterAssignmentRepo) CountDutyRequired(_ context.Context, semesterID string) (int64, error) {
	var count int64
	for _, a := range m.assignments {
		if a.SemesterID == semesterID && a.DutyRequired {
			count++
		}
	}
	return count, nil
}

func (m *mockUserSemesterAssignmentRepo) CountDutyRequiredSubmitted(_ context.Context, semesterID string) (int64, error) {
	var count int64
	for _, a := range m.assignments {
		if a.SemesterID == semesterID && a.DutyRequired && a.TimetableStatus == "submitted" {
			count++
		}
	}
	return count, nil
}

func (m *mockUserSemesterAssignmentRepo) GetByUserAndSemester(_ context.Context, userID, semesterID string) (*model.UserSemesterAssignment, error) {
	for i, a := range m.assignments {
		if a.UserID == userID && a.SemesterID == semesterID {
			return &m.assignments[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserSemesterAssignmentRepo) UpdateTimetableStatus(_ context.Context, assignmentID string, status string, submittedAt *time.Time, _ string) error {
	for i, a := range m.assignments {
		if a.AssignmentID == assignmentID {
			m.assignments[i].TimetableStatus = status
			m.assignments[i].TimetableSubmittedAt = submittedAt
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *mockUserSemesterAssignmentRepo) ListDutyRequiredBySemester(_ context.Context, semesterID string) ([]model.UserSemesterAssignment, error) {
	var result []model.UserSemesterAssignment
	for _, a := range m.assignments {
		if a.SemesterID == semesterID && a.DutyRequired {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockUserSemesterAssignmentRepo) ListDutyRequiredByDepartmentAndSemester(_ context.Context, departmentID, semesterID string) ([]model.UserSemesterAssignment, error) {
	var result []model.UserSemesterAssignment
	for _, a := range m.assignments {
		if a.SemesterID == semesterID && a.DutyRequired && a.User != nil && a.User.DepartmentID == departmentID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockUserSemesterAssignmentRepo) Create(_ context.Context, assignment *model.UserSemesterAssignment) error {
	if assignment.AssignmentID == "" {
		assignment.AssignmentID = fmt.Sprintf("assign-%d", len(m.assignments)+1)
	}
	m.assignments = append(m.assignments, *assignment)
	return nil
}

func (m *mockUserSemesterAssignmentRepo) BatchUpsert(_ context.Context, semesterID string, userIDs []string, dutyRequired bool, callerID string) error {
	for _, uid := range userIDs {
		found := false
		for i, a := range m.assignments {
			if a.UserID == uid && a.SemesterID == semesterID {
				m.assignments[i].DutyRequired = dutyRequired
				found = true
				break
			}
		}
		if !found {
			m.assignments = append(m.assignments, model.UserSemesterAssignment{
				AssignmentID: fmt.Sprintf("assign-%d", len(m.assignments)+1),
				UserID:       uid,
				SemesterID:   semesterID,
				DutyRequired: dutyRequired,
			})
		}
	}
	return nil
}

func (m *mockUserSemesterAssignmentRepo) ListByDepartmentAndSemester(_ context.Context, departmentID, semesterID string) ([]model.UserSemesterAssignment, error) {
	var result []model.UserSemesterAssignment
	for _, a := range m.assignments {
		if a.SemesterID == semesterID && a.User != nil && a.User.DepartmentID == departmentID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockUserSemesterAssignmentRepo) UpdateDutyRequired(_ context.Context, assignmentID string, dutyRequired bool, _ string) error {
	for i, a := range m.assignments {
		if a.AssignmentID == assignmentID {
			m.assignments[i].DutyRequired = dutyRequired
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

// ── Mock ScheduleRepository ──

type mockScheduleRepo struct {
	schedules map[string]*model.Schedule
	idCounter int
}

func newMockScheduleRepo() *mockScheduleRepo {
	return &mockScheduleRepo{schedules: make(map[string]*model.Schedule)}
}

func (m *mockScheduleRepo) Create(_ context.Context, schedule *model.Schedule) error {
	if schedule.ScheduleID == "" {
		m.idCounter++
		schedule.ScheduleID = fmt.Sprintf("sched-%d", m.idCounter)
	}
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()
	m.schedules[schedule.ScheduleID] = schedule
	return nil
}

func (m *mockScheduleRepo) GetByID(_ context.Context, id string) (*model.Schedule, error) {
	if s, ok := m.schedules[id]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockScheduleRepo) GetBySemester(_ context.Context, semesterID string) (*model.Schedule, error) {
	for _, s := range m.schedules {
		if s.SemesterID == semesterID && s.Status != "archived" {
			return s, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockScheduleRepo) GetLatestBySemester(_ context.Context, semesterID string) (*model.Schedule, error) {
	for _, s := range m.schedules {
		if s.SemesterID == semesterID {
			return s, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockScheduleRepo) Update(_ context.Context, schedule *model.Schedule) error {
	schedule.UpdatedAt = time.Now()
	m.schedules[schedule.ScheduleID] = schedule
	return nil
}

func (m *mockScheduleRepo) Delete(_ context.Context, id string) error {
	delete(m.schedules, id)
	return nil
}

// ── Mock ScheduleItemRepository ──

type mockScheduleItemRepo struct {
	items     map[string]*model.ScheduleItem
	idCounter int
}

func newMockScheduleItemRepo() *mockScheduleItemRepo {
	return &mockScheduleItemRepo{items: make(map[string]*model.ScheduleItem)}
}

func (m *mockScheduleItemRepo) BatchCreate(_ context.Context, items []model.ScheduleItem) error {
	for i := range items {
		m.idCounter++
		items[i].ScheduleItemID = fmt.Sprintf("item-%d", m.idCounter)
		items[i].CreatedAt = time.Now()
		items[i].UpdatedAt = time.Now()
		cp := items[i]
		m.items[cp.ScheduleItemID] = &cp
	}
	return nil
}

func (m *mockScheduleItemRepo) GetByID(_ context.Context, id string) (*model.ScheduleItem, error) {
	if item, ok := m.items[id]; ok {
		return item, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockScheduleItemRepo) ListBySchedule(_ context.Context, scheduleID string) ([]model.ScheduleItem, error) {
	var result []model.ScheduleItem
	for _, item := range m.items {
		if item.ScheduleID == scheduleID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockScheduleItemRepo) ListByScheduleAndMember(_ context.Context, scheduleID, memberID string) ([]model.ScheduleItem, error) {
	var result []model.ScheduleItem
	for _, item := range m.items {
		if item.ScheduleID == scheduleID && item.MemberID == memberID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *mockScheduleItemRepo) Update(_ context.Context, item *model.ScheduleItem) error {
	item.UpdatedAt = time.Now()
	m.items[item.ScheduleItemID] = item
	return nil
}

func (m *mockScheduleItemRepo) DeleteBySchedule(_ context.Context, scheduleID string) error {
	for id, item := range m.items {
		if item.ScheduleID == scheduleID {
			delete(m.items, id)
		}
	}
	return nil
}

// ── Mock ScheduleMemberSnapshotRepository ──

type mockScheduleMemberSnapshotRepo struct {
	snapshots []model.ScheduleMemberSnapshot
}

func newMockScheduleMemberSnapshotRepo() *mockScheduleMemberSnapshotRepo {
	return &mockScheduleMemberSnapshotRepo{}
}

func (m *mockScheduleMemberSnapshotRepo) BatchCreate(_ context.Context, snapshots []model.ScheduleMemberSnapshot) error {
	m.snapshots = append(m.snapshots, snapshots...)
	return nil
}

func (m *mockScheduleMemberSnapshotRepo) ListBySchedule(_ context.Context, scheduleID string) ([]model.ScheduleMemberSnapshot, error) {
	var result []model.ScheduleMemberSnapshot
	for _, s := range m.snapshots {
		if s.ScheduleID == scheduleID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockScheduleMemberSnapshotRepo) DeleteBySchedule(_ context.Context, scheduleID string) error {
	var remaining []model.ScheduleMemberSnapshot
	for _, s := range m.snapshots {
		if s.ScheduleID != scheduleID {
			remaining = append(remaining, s)
		}
	}
	m.snapshots = remaining
	return nil
}

// ── Mock ScheduleChangeLogRepository ──

type mockScheduleChangeLogRepo struct {
	logs []model.ScheduleChangeLog
}

func newMockScheduleChangeLogRepo() *mockScheduleChangeLogRepo {
	return &mockScheduleChangeLogRepo{}
}

func (m *mockScheduleChangeLogRepo) Create(_ context.Context, log *model.ScheduleChangeLog) error {
	m.logs = append(m.logs, *log)
	return nil
}

func (m *mockScheduleChangeLogRepo) ListBySchedule(_ context.Context, scheduleID string, offset, limit int) ([]model.ScheduleChangeLog, int64, error) {
	var filtered []model.ScheduleChangeLog
	for _, l := range m.logs {
		if l.ScheduleID == scheduleID {
			filtered = append(filtered, l)
		}
	}
	total := int64(len(filtered))
	if offset >= len(filtered) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}
