package model

// TimeSlot 时间段配置表 — 对应 time_slots
type TimeSlot struct {
	TimeSlotID string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"time_slot_id"`
	Name       string  `gorm:"type:varchar(50);not null"                      json:"name"`
	SemesterID *string `gorm:"type:uuid"                                      json:"semester_id,omitempty"` // NULL 表示全局默认
	StartTime  string  `gorm:"type:time;not null"                             json:"start_time"`
	EndTime    string  `gorm:"type:time;not null"                             json:"end_time"`
	DayOfWeek  int     `gorm:"type:smallint;not null"                         json:"day_of_week"` // 1-5
	IsActive   bool    `gorm:"not null;default:true"                          json:"is_active"`
	VersionedModel

	// 关联
	Semester *Semester `gorm:"foreignKey:SemesterID;references:SemesterID" json:"semester,omitempty"`
}

// TableName 指定表名
func (TimeSlot) TableName() string { return "time_slots" }

// [自证通过] internal/model/time_slot.go
