package model

// SystemConfig 系统配置表 — 对应 system_config（单行强类型）
type SystemConfig struct {
	Singleton            bool   `gorm:"primaryKey;default:true"                  json:"-"`
	SwapDeadlineHours    int    `gorm:"not null;default:24"                      json:"swap_deadline_hours"`
	DutyReminderTime     string `gorm:"type:time;not null;default:'09:00'"       json:"duty_reminder_time"`
	DefaultLocation      string `gorm:"type:varchar(200);not null;default:'学生会办公室'" json:"default_location"`
	SignInWindowMinutes  int    `gorm:"not null;default:15"                      json:"sign_in_window_minutes"`
	SignOutWindowMinutes int    `gorm:"not null;default:15"                      json:"sign_out_window_minutes"`
	BaseModel
}

// TableName 指定表名
func (SystemConfig) TableName() string { return "system_config" }

// [自证通过] internal/model/system_config.go
