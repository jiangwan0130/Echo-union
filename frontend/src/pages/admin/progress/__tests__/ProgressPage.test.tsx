import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import ProgressPage from '../ProgressPage';

// Mock stores — useAuthStore() 在组件中以 useAuthStore() 无 selector 调用
const mockAuthStore = {
  user: null as Record<string, unknown> | null,
  isAdmin: vi.fn(() => true),
  isLeader: vi.fn(() => false),
  isMember: vi.fn(() => false),
  hasRole: vi.fn(),
  isAuthenticated: true,
  loading: false,
  login: vi.fn(),
  logout: vi.fn(),
  fetchCurrentUser: vi.fn(),
  setUser: vi.fn(),
};

vi.mock('@/stores', () => ({
  useAuthStore: () => mockAuthStore,
}));

vi.mock('@/services', () => ({
  timetableApi: {
    getProgress: vi.fn(),
    getDepartmentProgress: vi.fn(),
  },
  showError: vi.fn(),
}));

import { timetableApi } from '@/services';

const mockGetProgress = vi.mocked(timetableApi.getProgress);
const mockGetDeptProgress = vi.mocked(timetableApi.getDepartmentProgress);

function makeSuccessResponse<T>(data: T) {
  return { data: { code: 0, message: 'ok', data } };
}

describe('ProgressPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAuthStore.user = { id: '1', name: 'Admin', role: 'admin', department: null };
    mockAuthStore.isAdmin.mockReturnValue(true);
    mockAuthStore.isLeader.mockReturnValue(false);
  });

  it('admin 视角应展示全局统计卡片', async () => {
    mockGetProgress.mockResolvedValue(
      makeSuccessResponse({
        total: 20,
        submitted: 15,
        progress: 75,
        departments: [
          {
            department_id: 'd1',
            department_name: '技术部',
            total: 10,
            submitted: 8,
            progress: 80,
          },
          {
            department_id: 'd2',
            department_name: '运营部',
            total: 10,
            submitted: 7,
            progress: 70,
          },
        ],
      }) as never,
    );

    render(<ProgressPage />);

    await waitFor(() => {
      expect(screen.getByText('20')).toBeInTheDocument();
    });
    expect(screen.getByText('15')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
    // 部门名称可见
    expect(screen.getByText('技术部')).toBeInTheDocument();
    expect(screen.getByText('运营部')).toBeInTheDocument();
  });

  it('admin 视角应展示部门列表', async () => {
    mockGetProgress.mockResolvedValue(
      makeSuccessResponse({
        total: 10,
        submitted: 10,
        progress: 100,
        departments: [
          {
            department_id: 'd1',
            department_name: '设计部',
            total: 5,
            submitted: 5,
            progress: 100,
          },
        ],
      }) as never,
    );

    render(<ProgressPage />);

    await waitFor(() => {
      expect(screen.getByText('设计部')).toBeInTheDocument();
    });
  });

  it('leader 视角应展示本部门数据', async () => {
    mockAuthStore.isAdmin.mockReturnValue(false);
    mockAuthStore.isLeader.mockReturnValue(true);
    mockAuthStore.user = {
      id: '2',
      name: 'Leader',
      role: 'leader',
      department: { id: 'd1', name: '技术部' },
    };

    mockGetDeptProgress.mockResolvedValue(
      makeSuccessResponse({
        department_id: 'd1',
        department_name: '技术部',
        total: 5,
        submitted: 3,
        progress: 60,
        members: [
          {
            user_id: 'u1',
            name: '张三',
            student_id: '2021001',
            timetable_status: 'submitted',
            submitted_at: '2026-02-20T10:30:00Z',
          },
          {
            user_id: 'u2',
            name: '李四',
            student_id: '2021002',
            timetable_status: 'not_submitted',
          },
        ],
      }) as never,
    );

    render(<ProgressPage />);

    await waitFor(() => {
      expect(screen.getByText('技术部')).toBeInTheDocument();
    });
    expect(screen.getByText('张三')).toBeInTheDocument();
    expect(screen.getByText('李四')).toBeInTheDocument();
    // 表格中通过 Tag 组件显示"已提交"/"未提交"
    expect(screen.getAllByText('已提交').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('未提交').length).toBeGreaterThanOrEqual(1);
  });
});
