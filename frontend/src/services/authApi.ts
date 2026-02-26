import api from './api';
import type {
  ApiResponse,
  LoginRequest,
  TokenResponse,
  ChangePasswordRequest,
  UserDetail,
} from '@/types';

export const authApi = {
  login: (data: LoginRequest) =>
    api.post<ApiResponse<TokenResponse>>('/auth/login', data),

  refreshToken: () =>
    api.post<ApiResponse<TokenResponse>>('/auth/refresh', {}),

  logout: () => api.post<ApiResponse<null>>('/auth/logout'),

  getCurrentUser: () => api.get<ApiResponse<UserDetail>>('/auth/me'),

  changePassword: (data: ChangePasswordRequest) =>
    api.put<ApiResponse<null>>('/auth/password', data),
};
