package model

// Notification 通知消息表 — 对应 notifications
type Notification struct {
	NotificationID string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"notification_id"`
	UserID         string  `gorm:"type:uuid;not null"                             json:"user_id"`
	Type           string  `gorm:"type:varchar(50);not null"                      json:"type"`
	Title          string  `gorm:"type:varchar(200);not null"                     json:"title"`
	Content        string  `gorm:"type:text;not null"                             json:"content"`
	IsRead         bool    `gorm:"not null;default:false"                         json:"is_read"`
	RelatedType    *string `gorm:"type:varchar(20)"                               json:"related_type,omitempty"` // schedule | schedule_item | swap_request | duty_record
	RelatedID      *string `gorm:"type:uuid"                                      json:"related_id,omitempty"`
	SoftDeleteModel
}

// TableName 指定表名
func (Notification) TableName() string { return "notifications" }

// NotificationPreference 通知偏好表 — 对应 notification_preferences（与 users 1:1）
type NotificationPreference struct {
	UserID             string `gorm:"type:uuid;primaryKey"  json:"user_id"`
	SchedulePublished  bool   `gorm:"not null;default:true" json:"schedule_published"`
	DutyReminder       bool   `gorm:"not null;default:true" json:"duty_reminder"`
	SwapNotification   bool   `gorm:"not null;default:true" json:"swap_notification"`
	AbsentNotification bool   `gorm:"not null;default:true" json:"absent_notification"`
	BaseModel
}

// TableName 指定表名
func (NotificationPreference) TableName() string { return "notification_preferences" }

// [自证通过] internal/model/notification.go
