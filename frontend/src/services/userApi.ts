import api from './api';
import type {
  ApiResponse,
  PaginatedResponse,
  UserInfo,
  UserDetail,
  UserListParams,
  UpdateUserRequest,
  AssignRoleRequest,
  ResetPasswordResponse,
  ImportUserResponse,
} from '@/types';

export const userApi = {
  getCurrentUser: () => api.get<ApiResponse<UserDetail>>('/users/me'),

  getUser: (id: string) => api.get<ApiResponse<UserDetail>>(`/users/${id}`),

  listUsers: (params?: UserListParams) =>
    api.get<PaginatedResponse<UserInfo>>('/users', { params }),

  updateUser: (id: string, data: UpdateUserRequest) =>
    api.put<ApiResponse<UserInfo>>(`/users/${id}`, data),

  deleteUser: (id: string) => api.delete<ApiResponse<null>>(`/users/${id}`),

  assignRole: (id: string, data: AssignRoleRequest) =>
    api.put<ApiResponse<UserInfo>>(`/users/${id}/role`, data),

  resetPassword: (id: string) =>
    api.post<ApiResponse<ResetPasswordResponse>>(
      `/users/${id}/reset-password`,
    ),

  importUsers: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return api.post<ApiResponse<ImportUserResponse>>('/users/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
};
