package dto

// ── 排班规则模块 DTO ──

// UpdateScheduleRuleRequest 更新排班规则请求
type UpdateScheduleRuleRequest struct {
	IsEnabled *bool `json:"is_enabled"`
}

// ScheduleRuleResponse 排班规则信息响应
type ScheduleRuleResponse struct {
	ID             string `json:"id"`
	RuleCode       string `json:"rule_code"`
	RuleName       string `json:"rule_name"`
	Description    string `json:"description,omitempty"`
	IsEnabled      bool   `json:"is_enabled"`
	IsConfigurable bool   `json:"is_configurable"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
