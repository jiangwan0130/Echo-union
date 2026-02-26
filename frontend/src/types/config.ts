// ── 时间段 ──

export interface TimeSlotInfo {
  id: string;
  name: string;
  semester_id?: string;
  semester?: { id: string; name: string };
  start_time: string;
  end_time: string;
  day_of_week: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateTimeSlotRequest {
  name: string;
  semester_id?: string;
  start_time: string;
  end_time: string;
  day_of_week: number;
}

export interface UpdateTimeSlotRequest {
  name?: string;
  start_time?: string;
  end_time?: string;
  day_of_week?: number;
  is_active?: boolean;
}

export interface TimeSlotListParams {
  semester_id?: string;
  day_of_week?: number;
}

// ── 地点 ──

export interface LocationInfo {
  id: string;
  name: string;
  address?: string;
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateLocationRequest {
  name: string;
  address?: string;
  is_default?: boolean;
}

export interface UpdateLocationRequest {
  name?: string;
  address?: string;
  is_default?: boolean;
  is_active?: boolean;
}

export interface LocationListParams {
  include_inactive?: boolean;
}

// ── 排班规则 ──

export interface ScheduleRuleInfo {
  id: string;
  rule_code: string;
  rule_name: string;
  description?: string;
  is_enabled: boolean;
  is_configurable: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpdateScheduleRuleRequest {
  is_enabled?: boolean;
}

// ── 系统配置 ──

export interface SystemConfigInfo {
  swap_deadline_hours: number;
  duty_reminder_time: string;
  default_location: string;
  sign_in_window_minutes: number;
  sign_out_window_minutes: number;
  updated_at: string;
}

export interface UpdateSystemConfigRequest {
  swap_deadline_hours?: number;
  duty_reminder_time?: string;
  default_location?: string;
  sign_in_window_minutes?: number;
  sign_out_window_minutes?: number;
}
