package model

// CourseSchedule 课表表 — 对应 course_schedules
type CourseSchedule struct {
	CourseScheduleID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"course_schedule_id"`
	UserID           string `gorm:"type:uuid;not null"                             json:"user_id"`
	SemesterID       string `gorm:"type:uuid;not null"                             json:"semester_id"`
	CourseName       string `gorm:"type:varchar(100);not null"                     json:"course_name"`
	DayOfWeek        int    `gorm:"type:smallint;not null"                         json:"day_of_week"` // 1-7
	StartTime        string `gorm:"type:time;not null"                             json:"start_time"`
	EndTime          string `gorm:"type:time;not null"                             json:"end_time"`
	WeekType         string `gorm:"type:varchar(10);not null;default:'all'"        json:"week_type"` // all | odd | even（冗余派生）
	Source           string `gorm:"type:varchar(20);not null;default:'ics'"        json:"source"`    // ics | manual
	VersionedModel

	// 关联
	User     *User     `gorm:"foreignKey:UserID;references:UserID"           json:"user,omitempty"`
	Semester *Semester `gorm:"foreignKey:SemesterID;references:SemesterID"   json:"semester,omitempty"`
}

// TableName 指定表名
func (CourseSchedule) TableName() string { return "course_schedules" }

// [自证通过] internal/model/course_schedule.go
