import api from './api';
import type {
  ApiResponse,
  PaginatedResponse,
  AutoScheduleRequest,
  AutoScheduleResponse,
  ScheduleInfo,
  UpdateScheduleItemRequest,
  UpdatePublishedItemRequest,
  ValidateCandidateRequest,
  ValidateCandidateResponse,
  CandidateInfo,
  ScheduleChangeLog,
  ChangeLogListParams,
  ScopeCheckResponse,
} from '@/types';

export const scheduleApi = {
  autoSchedule: (data: AutoScheduleRequest) =>
    api.post<ApiResponse<AutoScheduleResponse>>('/schedules/auto', data),

  getSchedule: (semesterId?: string) =>
    api.get<ApiResponse<ScheduleInfo>>('/schedules', {
      params: semesterId ? { semester_id: semesterId } : undefined,
    }),

  getMySchedule: (semesterId?: string) =>
    api.get<ApiResponse<ScheduleInfo>>('/schedules/my', {
      params: semesterId ? { semester_id: semesterId } : undefined,
    }),

  updateItem: (
    itemId: string,
    data: UpdateScheduleItemRequest,
  ) =>
    api.put<ApiResponse<null>>(
      `/schedules/items/${itemId}`,
      data,
    ),

  validateCandidate: (
    itemId: string,
    data: ValidateCandidateRequest,
  ) =>
    api.post<ApiResponse<ValidateCandidateResponse>>(
      `/schedules/items/${itemId}/validate`,
      data,
    ),

  getCandidates: (itemId: string) =>
    api.get<ApiResponse<CandidateInfo[]>>(
      `/schedules/items/${itemId}/candidates`,
    ),

  publish: () =>
    api.post<ApiResponse<ScheduleInfo>>(
      '/schedules/publish',
    ),

  updatePublishedItem: (
    itemId: string,
    data: UpdatePublishedItemRequest,
  ) =>
    api.put<ApiResponse<null>>(
      `/schedules/published/items/${itemId}`,
      data,
    ),

  listChangeLogs: (params: ChangeLogListParams) =>
    api.get<PaginatedResponse<ScheduleChangeLog>>('/schedules/change-logs', {
      params,
    }),

  checkScope: (scheduleId: string) =>
    api.post<ApiResponse<ScopeCheckResponse>>(
      `/schedules/${scheduleId}/scope-check`,
    ),
};
