export type SemesterPhase = 'configuring' | 'collecting' | 'scheduling' | 'published';

export interface SemesterInfo {
  id: string;
  name: string;
  start_date: string;
  end_date: string;
  first_week_type: 'odd' | 'even';
  is_active: boolean;
  status: string;
  phase: SemesterPhase;
  created_at: string;
  updated_at: string;
}

export interface SemesterBrief {
  id: string;
  name: string;
}

export interface CreateSemesterRequest {
  name: string;
  start_date: string;
  end_date: string;
  first_week_type: 'odd' | 'even';
}

export interface UpdateSemesterRequest {
  name?: string;
  start_date?: string;
  end_date?: string;
  first_week_type?: 'odd' | 'even';
  status?: 'active' | 'archived';
}

// ── 阶段推进 ──

export interface AdvancePhaseRequest {
  target_phase: SemesterPhase;
}

export interface PhaseCheckItem {
  label: string;
  passed: boolean;
  message?: string;
}

export interface PhaseCheckResponse {
  current_phase: SemesterPhase;
  can_advance: boolean;
  checks: PhaseCheckItem[];
}

// ── 值班人员 ──

export interface DutyMemberItem {
  user_id: string;
  name: string;
  student_id: string;
  department_id: string;
  department_name: string;
  duty_required: boolean;
}

export interface DutyMembersRequest {
  user_ids: string[];
}

// ── 待办通知 ──

export interface PendingTodoItem {
  type: string;
  title: string;
  message: string;
}
