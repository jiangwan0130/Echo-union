package model

import "time"

// UnavailableTime 不可用时间表 — 对应 unavailable_times
type UnavailableTime struct {
	UnavailableTimeID string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"unavailable_time_id"`
	UserID            string     `gorm:"type:uuid;not null"                             json:"user_id"`
	SemesterID        string     `gorm:"type:uuid;not null"                             json:"semester_id"`
	DayOfWeek         int        `gorm:"type:smallint;not null"                         json:"day_of_week"` // 1-7
	StartTime         string     `gorm:"type:time;not null"                             json:"start_time"`
	EndTime           string     `gorm:"type:time;not null"                             json:"end_time"`
	Reason            string     `gorm:"type:varchar(200)"                              json:"reason,omitempty"`
	RepeatType        string     `gorm:"type:varchar(20);not null;default:'weekly'"     json:"repeat_type"` // once | weekly
	SpecificDate      *time.Time `gorm:"type:date"                                      json:"specific_date,omitempty"`
	WeekType          string     `gorm:"type:varchar(10);not null;default:'all'"        json:"week_type"` // all | odd | even
	VersionedModel

	// 关联
	User     *User     `gorm:"foreignKey:UserID;references:UserID"         json:"user,omitempty"`
	Semester *Semester `gorm:"foreignKey:SemesterID;references:SemesterID" json:"semester,omitempty"`
}

// TableName 指定表名
func (UnavailableTime) TableName() string { return "unavailable_times" }

// [自证通过] internal/model/unavailable_time.go
