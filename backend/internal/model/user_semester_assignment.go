package model

import "time"

// UserSemesterAssignment 用户-学期分配表 — 对应 user_semester_assignments
type UserSemesterAssignment struct {
	AssignmentID         string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"assignment_id"`
	UserID               string     `gorm:"type:uuid;not null"                             json:"user_id"`
	SemesterID           string     `gorm:"type:uuid;not null"                             json:"semester_id"`
	DutyRequired         bool       `gorm:"not null;default:false"                         json:"duty_required"`
	TimetableStatus      string     `gorm:"type:varchar(20);not null;default:'not_submitted'" json:"timetable_status"` // not_submitted | submitted
	TimetableSubmittedAt *time.Time `json:"timetable_submitted_at,omitempty"`
	VersionedModel

	// 关联
	User     *User     `gorm:"foreignKey:UserID;references:UserID"           json:"user,omitempty"`
	Semester *Semester `gorm:"foreignKey:SemesterID;references:SemesterID"   json:"semester,omitempty"`
}

// TableName 指定表名
func (UserSemesterAssignment) TableName() string { return "user_semester_assignments" }

// [自证通过] internal/model/user_semester_assignment.go
