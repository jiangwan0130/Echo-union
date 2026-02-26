import { create } from 'zustand';
import type { UserDetail, UserRole } from '@/types';
import { authApi } from '@/services';
import { setAccessToken } from '@/services';

interface AuthState {
  user: UserDetail | null;
  isAuthenticated: boolean;
  loading: boolean;

  login: (
    studentId: string,
    password: string,
    rememberMe?: boolean,
  ) => Promise<void>;
  logout: () => Promise<void>;
  fetchCurrentUser: () => Promise<void>;
  setUser: (user: UserDetail | null) => void;

  hasRole: (role: UserRole) => boolean;
  isAdmin: () => boolean;
  isLeader: () => boolean;
  isMember: () => boolean;
}

export const useAuthStore = create<AuthState>((set, get) => {
  // 监听 Token 失效事件（由 api.ts 拦截器触发），清除登录态 → AuthGuard 自动重定向
  window.addEventListener('auth:unauthorized', () => {
    setAccessToken(null);
    set({ user: null, isAuthenticated: false, loading: false });
  });

  return {
    user: null,
    isAuthenticated: false,
    loading: true,

  login: async (studentId, password, rememberMe = false) => {
    const { data } = await authApi.login({
      student_id: studentId,
      password,
      remember_me: rememberMe,
    });
    setAccessToken(data.data.access_token);
    set({
      user: data.data.user as UserDetail,
      isAuthenticated: true,
    });
  },

  logout: async () => {
    try {
      await authApi.logout();
    } finally {
      setAccessToken(null);
      set({ user: null, isAuthenticated: false });
    }
  },

  fetchCurrentUser: async () => {
    try {
      set({ loading: true });
      const { data } = await authApi.getCurrentUser();
      set({ user: data.data, isAuthenticated: true });
    } catch {
      set({ user: null, isAuthenticated: false });
    } finally {
      set({ loading: false });
    }
  },

  setUser: (user) => set({ user, isAuthenticated: !!user }),

  hasRole: (role) => get().user?.role === role,
  isAdmin: () => get().user?.role === 'admin',
  isLeader: () => get().user?.role === 'leader',
  isMember: () => get().user?.role === 'member',
  };
});
