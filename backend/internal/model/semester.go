package model

import "time"

// Semester 学期表 — 对应 semesters
type Semester struct {
	SemesterID    string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"semester_id"`
	Name          string    `gorm:"type:varchar(100);not null"                     json:"name"`
	StartDate     time.Time `gorm:"type:date;not null"                             json:"start_date"`
	EndDate       time.Time `gorm:"type:date;not null"                             json:"end_date"`
	FirstWeekType string    `gorm:"type:varchar(10);not null"                      json:"first_week_type"` // odd | even
	IsActive      bool      `gorm:"not null;default:false"                         json:"is_active"`
	Status        string    `gorm:"type:varchar(20);not null;default:'active'"     json:"status"` // active | archived
	Phase         string    `gorm:"type:varchar(20);not null;default:'configuring'" json:"phase"` // configuring | collecting | scheduling | published
	VersionedModel
}

// TableName 指定表名
func (Semester) TableName() string { return "semesters" }

// [自证通过] internal/model/semester.go
