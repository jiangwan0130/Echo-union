import api from './api';
import type {
  ApiResponse,
  DepartmentDetail,
  DepartmentBrief,
  CreateDepartmentRequest,
  UpdateDepartmentRequest,
  DepartmentListParams,
  DepartmentMember,
  SetDutyMembersRequest,
  SetDutyMembersResponse,
} from '@/types';

export const departmentApi = {
  list: (params?: DepartmentListParams) =>
    api.get<ApiResponse<DepartmentDetail[]>>('/departments', { params }),

  getById: (id: string) =>
    api.get<ApiResponse<DepartmentDetail>>(`/departments/${id}`),

  create: (data: CreateDepartmentRequest) =>
    api.post<ApiResponse<DepartmentBrief>>('/departments', data),

  update: (id: string, data: UpdateDepartmentRequest) =>
    api.put<ApiResponse<DepartmentDetail>>(`/departments/${id}`, data),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/departments/${id}`),

  getMembers: (id: string, semesterId?: string) =>
    api.get<ApiResponse<DepartmentMember[]>>(`/departments/${id}/members`, {
      params: semesterId ? { semester_id: semesterId } : undefined,
    }),

  setDutyMembers: (id: string, data: SetDutyMembersRequest) =>
    api.put<ApiResponse<SetDutyMembersResponse>>(
      `/departments/${id}/duty-members`,
      data,
    ),
};
