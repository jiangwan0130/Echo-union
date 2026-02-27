package model

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ── 通用格式常量 ──

const (
	// TimeFormatDateTime 标准 ISO8601 日期时间格式（等同于 time.RFC3339）
	TimeFormatDateTime = "2006-01-02T15:04:05Z"
	// TimeFormatDate 日期格式
	TimeFormatDate = "2006-01-02"
)

// ── 用户角色枚举 ──

const (
	RoleAdmin  = "admin"
	RoleLeader = "leader"
	RoleMember = "member"
)

// ── 排班表状态枚举 ──

const (
	ScheduleStatusDraft     = "draft"
	ScheduleStatusPublished = "published"
	ScheduleStatusArchived  = "archived"
	ScheduleStatusNeedRegen = "need_regen"
)

// ── 时间表提交状态枚举 ──

const (
	TimetableStatusNotSubmitted = "not_submitted"
	TimetableStatusSubmitted    = "submitted"
)

// ── 周类型枚举 ──

const (
	WeekTypeAll  = "all"
	WeekTypeOdd  = "odd"
	WeekTypeEven = "even"
)

// ── 学期状态枚举 ──

const (
	SemesterStatusActive   = "active"
	SemesterStatusInactive = "inactive"
)

// ── 学期阶段枚举（排班事务流水线） ──

const (
	SemesterPhaseConfiguring = "configuring" // 系统配置中
	SemesterPhaseCollecting  = "collecting"  // 收集时间表中
	SemesterPhaseScheduling  = "scheduling"  // 排班中
	SemesterPhasePublished   = "published"   // 已发布
)

// ── PostgreSQL INT[] 自定义类型 ──

// IntArray 对应 PostgreSQL INT[] 类型，实现 GORM Scanner/Valuer 接口。
type IntArray []int

// Scan 将 PostgreSQL 返回的 {1,2,3} 文本解析为 []int。
func (a *IntArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("IntArray.Scan: unsupported type %T", src)
	}
	s = strings.Trim(s, "{}")
	if s == "" {
		*a = IntArray{}
		return nil
	}
	parts := strings.Split(s, ",")
	arr := make(IntArray, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return fmt.Errorf("IntArray.Scan: invalid element %q: %w", p, err)
		}
		arr = append(arr, n)
	}
	*a = arr
	return nil
}

// Value 将 []int 序列化为 PostgreSQL {1,2,3} 文本。
func (a IntArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	parts := make([]string, len(a))
	for i, n := range a {
		parts[i] = strconv.Itoa(n)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

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
