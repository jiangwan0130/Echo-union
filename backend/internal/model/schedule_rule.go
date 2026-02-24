package model

// ScheduleRule 排班规则配置表 — 对应 schedule_rules
type ScheduleRule struct {
	RuleID         string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"rule_id"`
	RuleCode       string `gorm:"type:varchar(20);not null"                      json:"rule_code"`
	RuleName       string `gorm:"type:varchar(100);not null"                     json:"rule_name"`
	Description    string `gorm:"type:varchar(500)"                              json:"description,omitempty"`
	IsEnabled      bool   `gorm:"not null;default:true"                          json:"is_enabled"`
	IsConfigurable bool   `gorm:"not null;default:true"                          json:"is_configurable"`
	VersionedModel
}

// TableName 指定表名
func (ScheduleRule) TableName() string { return "schedule_rules" }

// [自证通过] internal/model/schedule_rule.go
