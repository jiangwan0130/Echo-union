import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import PreCheckPanel from '../PreCheckPanel';

// Mock services
vi.mock('@/services', () => ({
  timetableApi: {
    getProgress: vi.fn(),
  },
  scheduleRuleApi: {
    list: vi.fn(),
  },
  scheduleApi: {
    checkScope: vi.fn(),
  },
  showError: vi.fn(),
}));

import { timetableApi, scheduleRuleApi, scheduleApi } from '@/services';

const mockGetProgress = vi.mocked(timetableApi.getProgress);
const mockRulesList = vi.mocked(scheduleRuleApi.list);
const mockCheckScope = vi.mocked(scheduleApi.checkScope);

function makeSuccessResponse<T>(data: T) {
  return { data: { code: 0, message: 'ok', data } };
}

describe('PreCheckPanel', () => {
  const onAllPassed = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    onAllPassed.mockReset();
  });

  it('无学期时应显示失败状态', async () => {
    render(<PreCheckPanel onAllPassed={onAllPassed} />);

    await waitFor(() => {
      expect(onAllPassed).toHaveBeenCalledWith(false);
    });
    expect(screen.getByText('未选择学期')).toBeInTheDocument();
  });

  it('所有检查通过时应调用 onAllPassed(true)', async () => {
    mockGetProgress.mockResolvedValue(
      makeSuccessResponse({
        total: 10,
        submitted: 10,
        progress: 100,
        departments: [],
      }) as never,
    );
    mockRulesList.mockResolvedValue(
      makeSuccessResponse([
        { id: 'r1', rule_name: 'rule1', is_enabled: true },
      ]) as never,
    );

    render(
      <PreCheckPanel semesterId="sem-1" onAllPassed={onAllPassed} />,
    );

    await waitFor(() => {
      expect(onAllPassed).toHaveBeenCalledWith(true);
    });
  });

  it('提交率不足100%时应显示失败', async () => {
    mockGetProgress.mockResolvedValue(
      makeSuccessResponse({
        total: 10,
        submitted: 5,
        progress: 50,
        departments: [],
      }) as never,
    );
    mockRulesList.mockResolvedValue(
      makeSuccessResponse([]) as never,
    );

    render(
      <PreCheckPanel semesterId="sem-1" onAllPassed={onAllPassed} />,
    );

    await waitFor(() => {
      expect(onAllPassed).toHaveBeenCalledWith(false);
    });
    expect(screen.getByText(/5人未提交/)).toBeInTheDocument();
  });

  it('带 scheduleId 时应执行 scope check', async () => {
    mockGetProgress.mockResolvedValue(
      makeSuccessResponse({
        total: 10,
        submitted: 10,
        progress: 100,
        departments: [],
      }) as never,
    );
    mockRulesList.mockResolvedValue(
      makeSuccessResponse([
        { id: 'r1', rule_name: 'rule1', is_enabled: true },
      ]) as never,
    );
    mockCheckScope.mockResolvedValue(
      makeSuccessResponse({
        changed: true,
        added_users: ['张三', '李四'],
        removed_users: [],
      }) as never,
    );

    render(
      <PreCheckPanel
        semesterId="sem-1"
        scheduleId="sch-1"
        onAllPassed={onAllPassed}
      />,
    );

    await waitFor(() => {
      expect(mockCheckScope).toHaveBeenCalledWith('sch-1');
    });
    // scope warn 不阻塞排班，应仍然 pass
    await waitFor(() => {
      expect(onAllPassed).toHaveBeenCalledWith(true);
    });
    expect(screen.getByText(/新增 2 人/)).toBeInTheDocument();
  });

  it('scope check 无变更时应显示通过', async () => {
    mockGetProgress.mockResolvedValue(
      makeSuccessResponse({
        total: 10,
        submitted: 10,
        progress: 100,
        departments: [],
      }) as never,
    );
    mockRulesList.mockResolvedValue(
      makeSuccessResponse([
        { id: 'r1', rule_name: 'rule1', is_enabled: true },
      ]) as never,
    );
    mockCheckScope.mockResolvedValue(
      makeSuccessResponse({ changed: false }) as never,
    );

    render(
      <PreCheckPanel
        semesterId="sem-1"
        scheduleId="sch-1"
        onAllPassed={onAllPassed}
      />,
    );

    await waitFor(() => {
      expect(screen.getByText(/无变更/)).toBeInTheDocument();
    });
  });
});
