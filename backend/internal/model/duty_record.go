package model

import "time"

// DutyRecord 值班记录表 — 对应 duty_records
type DutyRecord struct {
	DutyRecordID   string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"duty_record_id"`
	ScheduleItemID string     `gorm:"type:uuid;not null"                             json:"schedule_item_id"`
	MemberID       string     `gorm:"type:uuid;not null"                             json:"member_id"` // 冗余快照
	DutyDate       time.Time  `gorm:"type:date;not null"                             json:"duty_date"`
	Status         string     `gorm:"type:varchar(20);not null;default:'pending'"    json:"status"` // pending | on_duty | completed | absent | absent_made_up | no_sign_out
	SignInTime     *time.Time `json:"sign_in_time,omitempty"`
	SignOutTime    *time.Time `json:"sign_out_time,omitempty"`
	IsLate         bool       `gorm:"not null;default:false"                         json:"is_late"` // 冗余派生
	MakeUpTime     *time.Time `json:"make_up_time,omitempty"`
	VersionedModel

	// 关联
	ScheduleItem *ScheduleItem `gorm:"foreignKey:ScheduleItemID;references:ScheduleItemID" json:"schedule_item,omitempty"`
	Member       *User         `gorm:"foreignKey:MemberID;references:UserID"                json:"member,omitempty"`
}

// TableName 指定表名
func (DutyRecord) TableName() string { return "duty_records" }

// [自证通过] internal/model/duty_record.go
