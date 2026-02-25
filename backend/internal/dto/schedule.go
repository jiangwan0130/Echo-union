package dto

// ── 排班模块 DTO ──

// AutoScheduleRequest 自动排班请求
type AutoScheduleRequest struct {
	SemesterID string `json:"semester_id" binding:"required,uuid"`
}

// UpdateScheduleItemRequest 手动调整排班项请求
type UpdateScheduleItemRequest struct {
	MemberID   *string `json:"member_id"   binding:"omitempty,uuid"`
	LocationID *string `json:"location_id" binding:"omitempty,uuid"`
}

// PublishScheduleRequest 发布排班表请求
type PublishScheduleRequest struct {
	ScheduleID string `json:"schedule_id" binding:"required,uuid"`
}

// UpdatePublishedItemRequest 发布后修改排班项请求
type UpdatePublishedItemRequest struct {
	MemberID string `json:"member_id" binding:"required,uuid"`
	Reason   string `json:"reason"    binding:"required,min=2,max=500"`
}

// ValidateCandidateRequest 校验候选人是否可排请求
type ValidateCandidateRequest struct {
	MemberID string `json:"member_id" binding:"required,uuid"`
}

// ScheduleChangeLogListRequest 变更日志列表查询参数
type ScheduleChangeLogListRequest struct {
	ScheduleID string `form:"schedule_id" binding:"required,uuid"`
	PaginationRequest
}

// ── 响应 ──

// ScheduleResponse 排班表响应
type ScheduleResponse struct {
	ID          string                 `json:"id"`
	SemesterID  string                 `json:"semester_id"`
	Semester    *SemesterBrief         `json:"semester,omitempty"`
	Status      string                 `json:"status"`
	PublishedAt *string                `json:"published_at,omitempty"`
	Items       []ScheduleItemResponse `json:"items,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// ScheduleItemResponse 排班明细响应
type ScheduleItemResponse struct {
	ID         string         `json:"id"`
	ScheduleID string         `json:"schedule_id"`
	WeekNumber int            `json:"week_number"`
	TimeSlot   *TimeSlotBrief `json:"time_slot,omitempty"`
	Member     *MemberBrief   `json:"member,omitempty"`
	Location   *LocationBrief `json:"location,omitempty"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

// TimeSlotBrief 时间段简要信息
type TimeSlotBrief struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// MemberBrief 成员简要信息
type MemberBrief struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	StudentID  string              `json:"student_id"`
	Department *DepartmentResponse `json:"department,omitempty"`
}

// LocationBrief 地点简要信息
type LocationBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CandidateResponse 候选人响应
type CandidateResponse struct {
	UserID     string              `json:"user_id"`
	Name       string              `json:"name"`
	StudentID  string              `json:"student_id"`
	Department *DepartmentResponse `json:"department,omitempty"`
	Available  bool                `json:"available"`
	Conflicts  []string            `json:"conflicts,omitempty"` // 冲突原因列表
}

// ValidateCandidateResponse 候选人校验响应
type ValidateCandidateResponse struct {
	Valid     bool     `json:"valid"`
	Conflicts []string `json:"conflicts,omitempty"`
}

// ScheduleChangeLogResponse 变更日志响应
type ScheduleChangeLogResponse struct {
	ID                 string `json:"id"`
	ScheduleID         string `json:"schedule_id"`
	ScheduleItemID     string `json:"schedule_item_id"`
	OriginalMemberID   string `json:"original_member_id"`
	OriginalMemberName string `json:"original_member_name,omitempty"`
	NewMemberID        string `json:"new_member_id"`
	NewMemberName      string `json:"new_member_name,omitempty"`
	ChangeType         string `json:"change_type"`
	Reason             string `json:"reason,omitempty"`
	OperatorID         string `json:"operator_id"`
	CreatedAt          string `json:"created_at"`
}

// AutoScheduleResponse 自动排班结果响应
type AutoScheduleResponse struct {
	Schedule    *ScheduleResponse `json:"schedule"`
	TotalSlots  int               `json:"total_slots"`
	FilledSlots int               `json:"filled_slots"`
	Warnings    []string          `json:"warnings,omitempty"`
}

// ScopeCheckResponse 范围检测响应
type ScopeCheckResponse struct {
	Changed      bool     `json:"changed"`
	AddedUsers   []string `json:"added_users,omitempty"`
	RemovedUsers []string `json:"removed_users,omitempty"`
}
