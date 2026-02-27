import type { DepartmentBrief } from './department';
import type { PaginationParams } from './api';

// ── 用户信息 ──

export type UserRole = 'admin' | 'leader' | 'member';

export interface UserInfo {
  id: string;
  name: string;
  email: string;
  student_id: string;
  role: UserRole;
  department?: DepartmentBrief;
  must_change_password?: boolean;
}

export interface UserDetail extends UserInfo {
  must_change_password: boolean;
  created_at: string;
}

// ── 请求 ──

export interface UserListParams extends PaginationParams {
  department_id?: string;
  role?: UserRole;
  keyword?: string;
}

export interface CreateUserRequest {
  name: string;
  student_id: string;
  email: string;
  role: UserRole;
  department_id: string;
}

export interface CreateUserResponse {
  user: UserInfo;
  temp_password: string;
}

export interface UpdateUserRequest {
  name?: string;
  email?: string;
  department_id?: string;
}

export interface AssignRoleRequest {
  role: UserRole;
}

// ── 响应 ──

export interface ResetPasswordResponse {
  temp_password: string;
}

export interface ImportUserResponse {
  total: number;
  success: number;
  failed: number;
  errors?: ImportUserError[];
}

export interface ImportUserError {
  row: number;
  reason: string;
}
