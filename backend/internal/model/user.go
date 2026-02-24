package model

// User 用户表 — 对应 users
type User struct {
	UserID             string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"user_id"`
	Name               string `gorm:"type:varchar(100);not null"                     json:"name"`
	StudentID          string `gorm:"type:varchar(20);not null"                      json:"student_id"`
	Email              string `gorm:"type:varchar(255);not null"                     json:"email"`
	PasswordHash       string `gorm:"type:varchar(255);not null"                     json:"-"`
	Role               string `gorm:"type:varchar(20);not null;default:'member'"     json:"role"`
	DepartmentID       string `gorm:"type:uuid;not null"                             json:"department_id"`
	MustChangePassword bool   `gorm:"not null;default:false"                         json:"must_change_password"`
	VersionedModel

	// 关联
	Department *Department `gorm:"foreignKey:DepartmentID;references:DepartmentID" json:"department,omitempty"`
}

// TableName 指定表名
func (User) TableName() string { return "users" }

// [自证通过] internal/model/user.go
