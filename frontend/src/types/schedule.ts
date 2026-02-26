import type { DepartmentBrief } from './department';
import type { SemesterBrief } from './semester';
import type { PaginationParams } from './api';

export interface TimeSlotBrief {
  id: string;
  name: string;
  day_of_week: number;
  start_time: string;
  end_time: string;
}

export interface MemberBrief {
  id: string;
  name: string;
  student_id: string;
  department?: DepartmentBrief;
}

export interface LocationBrief {
  id: string;
  name: string;
}

// ── 排班 ──

export interface ScheduleInfo {
  id: string;
  semester_id: string;
  semester?: SemesterBrief;
  status: string;
  published_at?: string;
  items?: ScheduleItem[];
  created_at: string;
  updated_at: string;
}

export interface ScheduleItem {
  id: string;
  schedule_id: string;
  week_number: number;
  time_slot?: TimeSlotBrief;
  member?: MemberBrief;
  location?: LocationBrief;
  created_at: string;
  updated_at: string;
}

// ── 请求 ──

export interface AutoScheduleRequest {
  semester_id: string;
}

export interface UpdateScheduleItemRequest {
  member_id?: string;
  location_id?: string;
}

export interface UpdatePublishedItemRequest {
  member_id: string;
  reason: string;
}

export interface ValidateCandidateRequest {
  member_id: string;
}

export interface ChangeLogListParams extends PaginationParams {
  schedule_id: string;
}

// ── 响应 ──

export interface AutoScheduleResponse {
  schedule: ScheduleInfo;
  total_slots: number;
  filled_slots: number;
  warnings?: string[];
}

export interface CandidateInfo {
  user_id: string;
  name: string;
  student_id: string;
  department?: DepartmentBrief;
  available: boolean;
  conflicts?: string[];
}

export interface ValidateCandidateResponse {
  valid: boolean;
  conflicts?: string[];
}

export interface ScheduleChangeLog {
  id: string;
  schedule_id: string;
  schedule_item_id: string;
  original_member_id: string;
  original_member_name?: string;
  new_member_id: string;
  new_member_name?: string;
  change_type: string;
  reason?: string;
  operator_id: string;
  created_at: string;
}

export interface ScopeCheckResponse {
  changed: boolean;
  added_users?: string[];
  removed_users?: string[];
}
