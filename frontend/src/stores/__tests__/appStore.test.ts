import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useAppStore } from '../appStore';

vi.mock('@/services', () => ({
  semesterApi: {
    getCurrent: vi.fn(),
  },
}));

import { semesterApi } from '@/services';

const mockGetCurrent = vi.mocked(semesterApi.getCurrent);

describe('appStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAppStore.setState({
      currentSemester: null,
      sidebarCollapsed: false,
      globalLoading: false,
    });
  });

  describe('fetchCurrentSemester', () => {
    it('获取成功时应设置当前学期', async () => {
      const mockSemester = {
        id: 'sem-1',
        name: '2025-2026 第二学期',
        start_date: '2026-02-16',
        end_date: '2026-07-10',
        is_current: true,
      };

      mockGetCurrent.mockResolvedValue({
        data: { code: 0, message: 'ok', data: mockSemester },
      } as never);

      await useAppStore.getState().fetchCurrentSemester();

      const state = useAppStore.getState();
      expect(state.currentSemester?.name).toBe('2025-2026 第二学期');
    });

    it('获取失败时应清除学期', async () => {
      useAppStore.setState({
        currentSemester: { id: 'old', name: 'Old' } as never,
      });
      mockGetCurrent.mockRejectedValue(new Error('Network'));

      await useAppStore.getState().fetchCurrentSemester();
      expect(useAppStore.getState().currentSemester).toBeNull();
    });
  });

  describe('setSidebarCollapsed', () => {
    it('应切换侧边栏折叠状态', () => {
      useAppStore.getState().setSidebarCollapsed(true);
      expect(useAppStore.getState().sidebarCollapsed).toBe(true);

      useAppStore.getState().setSidebarCollapsed(false);
      expect(useAppStore.getState().sidebarCollapsed).toBe(false);
    });
  });

  describe('setGlobalLoading', () => {
    it('应设置全局 loading 状态', () => {
      useAppStore.getState().setGlobalLoading(true);
      expect(useAppStore.getState().globalLoading).toBe(true);
    });
  });
});
