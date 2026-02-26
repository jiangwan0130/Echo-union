import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useAuthStore } from '../authStore';

// Mock services
vi.mock('@/services', () => ({
  authApi: {
    login: vi.fn(),
    logout: vi.fn(),
    getCurrentUser: vi.fn(),
  },
  setAccessToken: vi.fn(),
}));

import { authApi, setAccessToken } from '@/services';

const mockLogin = vi.mocked(authApi.login);
const mockLogout = vi.mocked(authApi.logout);
const mockGetCurrentUser = vi.mocked(authApi.getCurrentUser);
const mockSetToken = vi.mocked(setAccessToken);

describe('authStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      loading: true,
    });
  });

  describe('login', () => {
    it('应当在登录成功后设置 token 和用户信息', async () => {
      const mockUser = {
        id: '1',
        name: '张三',
        email: 'zhangsan@example.com',
        student_id: '2021001',
        role: 'admin' as const,
        must_change_password: false,
        created_at: '2026-01-01T00:00:00Z',
      };

      mockLogin.mockResolvedValue({
        data: {
          code: 0,
          message: 'success',
          data: {
            access_token: 'test-token',
            expires_in: 900,
            user: mockUser,
          },
        },
      } as never);

      await useAuthStore.getState().login('2021001', 'password123', false);

      expect(mockLogin).toHaveBeenCalledWith({
        student_id: '2021001',
        password: 'password123',
        remember_me: false,
      });
      expect(mockSetToken).toHaveBeenCalledWith('test-token');

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user?.name).toBe('张三');
    });

    it('登录失败时不应修改状态', async () => {
      mockLogin.mockRejectedValue(new Error('Invalid credentials'));

      await expect(
        useAuthStore.getState().login('wrong', 'wrong'),
      ).rejects.toThrow();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });
  });

  describe('logout', () => {
    it('应当清除用户状态和 token', async () => {
      // 先设置已登录状态
      useAuthStore.setState({
        user: { id: '1', name: 'Test', role: 'admin' } as never,
        isAuthenticated: true,
      });

      mockLogout.mockResolvedValue({ data: { code: 0, message: 'ok', data: null } } as never);

      await useAuthStore.getState().logout();

      expect(mockSetToken).toHaveBeenCalledWith(null);
      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });

    it('即使 logout API 失败也应清除本地状态', async () => {
      useAuthStore.setState({
        user: { id: '1', name: 'Test', role: 'admin' } as never,
        isAuthenticated: true,
      });

      mockLogout.mockRejectedValue(new Error('Network error'));

      // logout 内部使用 try/finally，异常会在 finally 清理后继续抛出
      try {
        await useAuthStore.getState().logout();
      } catch {
        // expected
      }

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
    });
  });

  describe('fetchCurrentUser', () => {
    it('获取成功时应设置用户信息和认证状态', async () => {
      const mockUser = {
        id: '1',
        name: '李四',
        email: 'lisi@example.com',
        student_id: '2021002',
        role: 'member' as const,
        must_change_password: false,
        created_at: '2026-01-01T00:00:00Z',
      };

      mockGetCurrentUser.mockResolvedValue({
        data: { code: 0, message: 'ok', data: mockUser },
      } as never);

      await useAuthStore.getState().fetchCurrentUser();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user?.name).toBe('李四');
      expect(state.loading).toBe(false);
    });

    it('获取失败时应清除状态', async () => {
      mockGetCurrentUser.mockRejectedValue(new Error('401'));

      await useAuthStore.getState().fetchCurrentUser();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.user).toBeNull();
      expect(state.loading).toBe(false);
    });
  });

  describe('hasRole / isAdmin / isLeader / isMember', () => {
    it('应基于用户角色返回正确结果', () => {
      useAuthStore.setState({
        user: { id: '1', name: 'Admin', role: 'admin' } as never,
        isAuthenticated: true,
      });

      const state = useAuthStore.getState();
      expect(state.hasRole('admin')).toBe(true);
      expect(state.hasRole('member')).toBe(false);
      expect(state.isAdmin()).toBe(true);
      expect(state.isLeader()).toBe(false);
      expect(state.isMember()).toBe(false);
    });

    it('用户为空时所有角色检查应返回 false', () => {
      const state = useAuthStore.getState();
      expect(state.hasRole('admin')).toBe(false);
      expect(state.isAdmin()).toBe(false);
    });
  });
});
