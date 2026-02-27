import api from './api';
import type {
  ApiResponse,
  SemesterInfo,
  CreateSemesterRequest,
  UpdateSemesterRequest,
  PhaseCheckResponse,
  AdvancePhaseRequest,
  DutyMemberItem,
  DutyMembersRequest,
  PendingTodoItem,
  TimeSlotInfo,
  CreateTimeSlotRequest,
  UpdateTimeSlotRequest,
  TimeSlotListParams,
  LocationInfo,
  CreateLocationRequest,
  UpdateLocationRequest,
  LocationListParams,
  ScheduleRuleInfo,
  UpdateScheduleRuleRequest,
  SystemConfigInfo,
  UpdateSystemConfigRequest,
} from '@/types';

// ── 学期 ──

export const semesterApi = {
  list: () => api.get<ApiResponse<SemesterInfo[]>>('/semesters'),

  getById: (id: string) =>
    api.get<ApiResponse<SemesterInfo>>(`/semesters/${id}`),

  getCurrent: () =>
    api.get<ApiResponse<SemesterInfo>>('/semesters/current'),

  create: (data: CreateSemesterRequest) =>
    api.post<ApiResponse<SemesterInfo>>('/semesters', data),

  update: (id: string, data: UpdateSemesterRequest) =>
    api.put<ApiResponse<SemesterInfo>>(`/semesters/${id}`, data),

  activate: (id: string) =>
    api.put<ApiResponse<SemesterInfo>>(`/semesters/${id}/activate`),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/semesters/${id}`),

  // ── 阶段推进 ──

  checkPhase: (id: string) =>
    api.get<ApiResponse<PhaseCheckResponse>>(`/semesters/${id}/phase-check`),

  advancePhase: (id: string, data: AdvancePhaseRequest) =>
    api.put<ApiResponse<null>>(`/semesters/${id}/phase`, data),

  // ── 值班人员管理 ──

  getDutyMembers: (id: string) =>
    api.get<ApiResponse<{ list: DutyMemberItem[] }>>(`/semesters/${id}/duty-members`),

  setDutyMembers: (id: string, data: DutyMembersRequest) =>
    api.put<ApiResponse<null>>(`/semesters/${id}/duty-members`, data),
};

// ── 待办通知 ──

export const notificationApi = {
  getPending: () =>
    api.get<ApiResponse<{ list: PendingTodoItem[] }>>('/notifications/pending'),
};

// ── 时间段 ──

export const timeSlotApi = {
  list: (params?: TimeSlotListParams) =>
    api.get<ApiResponse<TimeSlotInfo[]>>('/time-slots', { params }),

  getById: (id: string) =>
    api.get<ApiResponse<TimeSlotInfo>>(`/time-slots/${id}`),

  create: (data: CreateTimeSlotRequest) =>
    api.post<ApiResponse<TimeSlotInfo>>('/time-slots', data),

  update: (id: string, data: UpdateTimeSlotRequest) =>
    api.put<ApiResponse<TimeSlotInfo>>(`/time-slots/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/time-slots/${id}`),
};

// ── 地点 ──

export const locationApi = {
  list: (params?: LocationListParams) =>
    api.get<ApiResponse<LocationInfo[]>>('/locations', { params }),

  getById: (id: string) =>
    api.get<ApiResponse<LocationInfo>>(`/locations/${id}`),

  create: (data: CreateLocationRequest) =>
    api.post<ApiResponse<LocationInfo>>('/locations', data),

  update: (id: string, data: UpdateLocationRequest) =>
    api.put<ApiResponse<LocationInfo>>(`/locations/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/locations/${id}`),
};

// ── 排班规则 ──

export const scheduleRuleApi = {
  list: () => api.get<ApiResponse<ScheduleRuleInfo[]>>('/schedule-rules'),

  getById: (id: string) =>
    api.get<ApiResponse<ScheduleRuleInfo>>(`/schedule-rules/${id}`),

  update: (id: string, data: UpdateScheduleRuleRequest) =>
    api.put<ApiResponse<ScheduleRuleInfo>>(`/schedule-rules/${id}`, data),
};

// ── 系统配置 ──

export const systemConfigApi = {
  get: () => api.get<ApiResponse<SystemConfigInfo>>('/system-config'),

  update: (data: UpdateSystemConfigRequest) =>
    api.put<ApiResponse<SystemConfigInfo>>('/system-config', data),
};
