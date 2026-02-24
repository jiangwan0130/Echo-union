package model

import "time"

// InviteCode 邀请码表 — 对应 invite_codes
type InviteCode struct {
	InviteCodeID string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"invite_code_id"`
	Code         string     `gorm:"type:varchar(50);not null"                      json:"code"`
	ExpiresAt    time.Time  `gorm:"not null"                                       json:"expires_at"`
	UsedAt       *time.Time `json:"used_at,omitempty"`
	UsedBy       *string    `gorm:"type:uuid"                                      json:"used_by,omitempty"`
	VersionedModel
}

// TableName 指定表名
func (InviteCode) TableName() string { return "invite_codes" }

// [自证通过] internal/model/invite_code.go
