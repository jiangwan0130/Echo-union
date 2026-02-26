import type { UserInfo } from './user';

// ── 请求 ──

export interface LoginRequest {
  student_id: string;
  password: string;
  remember_me?: boolean;
}

export interface RefreshTokenRequest {
  refresh_token?: string;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

// ── 响应 ──

export interface TokenResponse {
  access_token: string;
  refresh_token?: string;
  expires_in: number;
  user: UserInfo;
}
