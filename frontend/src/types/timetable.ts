export interface CourseInfo {
  id: string;
  name: string;
  day_of_week: number;
  start_time: string;
  end_time: string;
  week_type: string;
  weeks: number[];
  source: string;
}

export interface UnavailableTime {
  id: string;
  day_of_week: number;
  start_time: string;
  end_time: string;
  reason: string;
  repeat_type: string;
  specific_date?: string;
  week_type: string;
}

export interface MyTimetableResponse {
  courses: CourseInfo[];
  unavailable: UnavailableTime[];
  submit_status: string;
  submitted_at?: string;
}

// ── 请求 ──

export interface ImportICSRequest {
  url?: string;
  semester_id?: string;
}

export interface CreateUnavailableTimeRequest {
  day_of_week: number;
  start_time: string;
  end_time: string;
  reason?: string;
  repeat_type?: 'weekly' | 'biweekly' | 'once';
  specific_date?: string;
  week_type?: 'all' | 'odd' | 'even';
  semester_id?: string;
}

export interface UpdateUnavailableTimeRequest {
  day_of_week?: number;
  start_time?: string;
  end_time?: string;
  reason?: string;
  repeat_type?: 'weekly' | 'biweekly' | 'once';
  specific_date?: string;
  week_type?: 'all' | 'odd' | 'even';
}

export interface SubmitTimetableRequest {
  semester_id?: string;
}

// ── 响应 ──

export interface ImportICSResponse {
  imported_count: number;
  events: ImportedCourseEvent[];
}

export interface ImportedCourseEvent {
  name: string;
  day_of_week: number;
  start_time: string;
  end_time: string;
  weeks: number[];
}

export interface SubmitTimetableResponse {
  submit_status: string;
  submitted_at?: string;
}

export interface TimetableProgressResponse {
  total: number;
  submitted: number;
  progress: number;
  departments: DepartmentProgressItem[];
}

export interface DepartmentProgressItem {
  department_id: string;
  department_name: string;
  total: number;
  submitted: number;
  progress: number;
}

export interface DepartmentProgressResponse {
  department_id: string;
  department_name: string;
  total: number;
  submitted: number;
  progress: number;
  members: DepartmentMemberStatus[];
}

export interface DepartmentMemberStatus {
  user_id: string;
  name: string;
  student_id: string;
  timetable_status: string;
  submitted_at?: string;
}
