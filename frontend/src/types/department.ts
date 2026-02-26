// ── 简要信息 ──

export interface DepartmentBrief {
  id: string;
  name: string;
}

// ── 详细信息 ──

export interface DepartmentDetail {
  id: string;
  name: string;
  description?: string;
  is_active: boolean;
  member_count: number;
  created_at: string;
  updated_at: string;
}

// ── 请求 ──

export interface CreateDepartmentRequest {
  name: string;
  description?: string;
}

export interface UpdateDepartmentRequest {
  name?: string;
  description?: string;
  is_active?: boolean;
}

export interface DepartmentListParams {
  include_inactive?: boolean;
}

// ── 成员 ──

export interface DepartmentMember {
  user_id: string;
  name: string;
  student_id: string;
  email: string;
  role: string;
  duty_required: boolean;
  timetable_status: string;
}

export interface SetDutyMembersRequest {
  semester_id: string;
  user_ids: string[];
}

export interface SetDutyMembersResponse {
  department_id: string;
  department_name: string;
  semester_id: string;
  total_set: number;
}
