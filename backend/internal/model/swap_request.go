package model

import "time"

// SwapRequest 换班申请表 — 对应 swap_requests
type SwapRequest struct {
	SwapRequestID     string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"swap_request_id"`
	ScheduleItemID    string     `gorm:"type:uuid;not null"                             json:"schedule_item_id"`
	ApplicantID       string     `gorm:"type:uuid;not null"                             json:"applicant_id"`
	TargetMemberID    string     `gorm:"type:uuid;not null"                             json:"target_member_id"`
	Reason            string     `gorm:"type:varchar(500)"                              json:"reason,omitempty"`
	Status            string     `gorm:"type:varchar(20);not null;default:'pending'"    json:"status"` // pending | reviewing | completed | rejected | cancelled
	TargetRespondedAt *time.Time `json:"target_responded_at,omitempty"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty"`
	ApprovedBy        *string    `gorm:"type:uuid"                                      json:"approved_by,omitempty"`
	RejectReason      string     `gorm:"type:varchar(500)"                              json:"reject_reason,omitempty"`
	VersionedModel

	// 关联
	ScheduleItem *ScheduleItem `gorm:"foreignKey:ScheduleItemID;references:ScheduleItemID" json:"schedule_item,omitempty"`
	Applicant    *User         `gorm:"foreignKey:ApplicantID;references:UserID"             json:"applicant,omitempty"`
	TargetMember *User         `gorm:"foreignKey:TargetMemberID;references:UserID"          json:"target_member,omitempty"`
}

// TableName 指定表名
func (SwapRequest) TableName() string { return "swap_requests" }

// [自证通过] internal/model/swap_request.go
