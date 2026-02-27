import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { ApiResponse } from '@/types';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
});

// ── Token 管理 ──

let accessToken: string | null = null;

export const setAccessToken = (token: string | null) => {
  accessToken = token;
};

export const getAccessToken = () => accessToken;

// ── 请求拦截器 ──

api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

// ── 响应拦截器（含 Token 自动刷新） ──

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}> = [];

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token!);
    }
  });
  failedQueue = [];
};

api.interceptors.response.use(
  (response) => {
    // 统一适配列表接口返回格式：如果 data.data 是含 list 属性的对象，自动展开为数组
    const body = response.data;
    if (
      body &&
      typeof body === 'object' &&
      'data' in body &&
      body.data !== null &&
      typeof body.data === 'object' &&
      !Array.isArray(body.data) &&
      Array.isArray((body.data as Record<string, unknown>).list)
    ) {
      body.data = (body.data as Record<string, unknown>).list;
    }
    return response;
  },
  async (error: AxiosError<ApiResponse>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // 401 且非刷新请求本身 且内存中有 accessToken（说明是 token 过期而非未登录）→ 尝试刷新
    if (
      error.response?.status === 401 &&
      !originalRequest._retry &&
      !originalRequest.url?.includes('/auth/refresh') &&
      accessToken !== null
    ) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({
            resolve: (token: string) => {
              originalRequest.headers.Authorization = `Bearer ${token}`;
              resolve(api(originalRequest));
            },
            reject,
          });
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const { data } = await api.post<
          ApiResponse<{ access_token: string; expires_in: number }>
        >('/auth/refresh', {});
        const newToken = data.data.access_token;
        setAccessToken(newToken);
        processQueue(null, newToken);
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      } catch (refreshError) {
        processQueue(refreshError, null);
        setAccessToken(null);
        // 触发自定义事件，由 authStore 响应处理（避免硬重载导致无限循环）
        window.dispatchEvent(new CustomEvent('auth:unauthorized'));
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  },
);

export default api;
