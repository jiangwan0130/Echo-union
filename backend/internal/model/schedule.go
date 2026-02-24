package model

import "time"

// Schedule 排班表 — 对应 schedules
type Schedule struct {
	ScheduleID  string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"schedule_id"`
	SemesterID  string     `gorm:"type:uuid;not null"                             json:"semester_id"`
	Status      string     `gorm:"type:varchar(20);not null;default:'draft'"      json:"status"` // draft | published | need_regen | archived
	PublishedAt *time.Time `json:"published_at,omitempty"`
	VersionedModel

	// 关联
	Semester *Semester      `gorm:"foreignKey:SemesterID;references:SemesterID" json:"semester,omitempty"`
	Items    []ScheduleItem `gorm:"foreignKey:ScheduleID"                       json:"items,omitempty"`
}

func (Schedule) TableName() string { return "schedules" }

// ScheduleItem 排班明细表 — 对应 schedule_items
type ScheduleItem struct {
	ScheduleItemID string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"schedule_item_id"`
	ScheduleID     string  `gorm:"type:uuid;not null"                             json:"schedule_id"`
	WeekNumber     int     `gorm:"type:smallint;not null"                         json:"week_number"` // 1 | 2
	TimeSlotID     string  `gorm:"type:uuid;not null"                             json:"time_slot_id"`
	MemberID       string  `gorm:"type:uuid;not null"                             json:"member_id"`
	LocationID     *string `gorm:"type:uuid"                                      json:"location_id,omitempty"`
	VersionedModel

	// 关联
	TimeSlot *TimeSlot `gorm:"foreignKey:TimeSlotID;references:TimeSlotID" json:"time_slot,omitempty"`
	Member   *User     `gorm:"foreignKey:MemberID;references:UserID"       json:"member,omitempty"`
	Location *Location `gorm:"foreignKey:LocationID;references:LocationID" json:"location,omitempty"`
}

func (ScheduleItem) TableName() string { return "schedule_items" }

// ScheduleMemberSnapshot 排班成员快照表 — 对应 schedule_member_snapshots
type ScheduleMemberSnapshot struct {
	SnapshotID   string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"snapshot_id"`
	ScheduleID   string    `gorm:"type:uuid;not null"                             json:"schedule_id"`
	UserID       string    `gorm:"type:uuid;not null"                             json:"user_id"`
	DepartmentID string    `gorm:"type:uuid;not null"                             json:"department_id"`
	SnapshotAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"             json:"snapshot_at"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"             json:"created_at"`
}

func (ScheduleMemberSnapshot) TableName() string { return "schedule_member_snapshots" }

// ScheduleChangeLog 排班变更记录表 — 对应 schedule_change_logs（纯审计日志）
type ScheduleChangeLog struct {
	ChangeLogID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"change_log_id"`
	ScheduleID         string    `gorm:"type:uuid;not null"                             json:"schedule_id"`
	ScheduleItemID     string    `gorm:"type:uuid;not null"                             json:"schedule_item_id"`
	OriginalMemberID   string    `gorm:"type:uuid;not null"                             json:"original_member_id"`
	NewMemberID        string    `gorm:"type:uuid;not null"                             json:"new_member_id"`
	OriginalTimeSlotID *string   `gorm:"type:uuid"                                      json:"original_time_slot_id,omitempty"`
	NewTimeSlotID      *string   `gorm:"type:uuid"                                      json:"new_time_slot_id,omitempty"`
	ChangeType         string    `gorm:"type:varchar(20);not null"                      json:"change_type"` // manual_adjust | swap | admin_modify
	Reason             string    `gorm:"type:varchar(500)"                              json:"reason,omitempty"`
	OperatorID         string    `gorm:"type:uuid;not null"                             json:"operator_id"`
	CreatedAt          time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"             json:"created_at"`
}

func (ScheduleChangeLog) TableName() string { return "schedule_change_logs" }

// [自证通过] internal/model/schedule.go
