package dto

// ── 系统配置模块 DTO ──

// UpdateSystemConfigRequest 更新系统配置请求
type UpdateSystemConfigRequest struct {
	SwapDeadlineHours    *int    `json:"swap_deadline_hours"     binding:"omitempty,min=1,max=168"`
	DutyReminderTime     *string `json:"duty_reminder_time"`
	DefaultLocation      *string `json:"default_location"        binding:"omitempty,min=1,max=200"`
	SignInWindowMinutes  *int    `json:"sign_in_window_minutes"  binding:"omitempty,min=1,max=60"`
	SignOutWindowMinutes *int    `json:"sign_out_window_minutes" binding:"omitempty,min=1,max=60"`
}

// SystemConfigResponse 系统配置响应
type SystemConfigResponse struct {
	SwapDeadlineHours    int    `json:"swap_deadline_hours"`
	DutyReminderTime     string `json:"duty_reminder_time"`
	DefaultLocation      string `json:"default_location"`
	SignInWindowMinutes  int    `json:"sign_in_window_minutes"`
	SignOutWindowMinutes int    `json:"sign_out_window_minutes"`
	UpdatedAt            string `json:"updated_at"`
}
