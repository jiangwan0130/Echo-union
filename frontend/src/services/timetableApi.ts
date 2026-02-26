import api from './api';
import type {
  ApiResponse,
  ImportICSRequest,
  ImportICSResponse,
  MyTimetableResponse,
  CreateUnavailableTimeRequest,
  UpdateUnavailableTimeRequest,
  UnavailableTime,
  SubmitTimetableRequest,
  SubmitTimetableResponse,
  TimetableProgressResponse,
  DepartmentProgressResponse,
} from '@/types';

export const timetableApi = {
  importICS: (data: ImportICSRequest | FormData) => {
    if (data instanceof FormData) {
      return api.post<ApiResponse<ImportICSResponse>>(
        '/timetables/import',
        data,
        { headers: { 'Content-Type': 'multipart/form-data' } },
      );
    }
    return api.post<ApiResponse<ImportICSResponse>>('/timetables/import', data);
  },

  getMyTimetable: (semesterId?: string) =>
    api.get<ApiResponse<MyTimetableResponse>>('/timetables/me', {
      params: semesterId ? { semester_id: semesterId } : undefined,
    }),

  createUnavailableTime: (data: CreateUnavailableTimeRequest) =>
    api.post<ApiResponse<UnavailableTime>>('/timetables/unavailable', data),

  updateUnavailableTime: (id: string, data: UpdateUnavailableTimeRequest) =>
    api.put<ApiResponse<UnavailableTime>>(
      `/timetables/unavailable/${id}`,
      data,
    ),

  deleteUnavailableTime: (id: string) =>
    api.delete<ApiResponse<null>>(`/timetables/unavailable/${id}`),

  submitTimetable: (data?: SubmitTimetableRequest) =>
    api.post<ApiResponse<SubmitTimetableResponse>>(
      '/timetables/submit',
      data ?? {},
    ),

  getProgress: () =>
    api.get<ApiResponse<TimetableProgressResponse>>('/timetables/progress'),

  getDepartmentProgress: (departmentId: string) =>
    api.get<ApiResponse<DepartmentProgressResponse>>(
      `/timetables/progress/department/${departmentId}`,
    ),
};
