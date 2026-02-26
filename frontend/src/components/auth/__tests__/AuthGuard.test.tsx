import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

// Mock stores — 需要返回完整的 Zustand hook 接口
const mockAuthState = {
  isAuthenticated: false,
  loading: false,
  user: null as Record<string, unknown> | null,
  login: vi.fn(),
  logout: vi.fn(),
  fetchCurrentUser: vi.fn(),
  setUser: vi.fn(),
  hasRole: vi.fn(),
  isAdmin: vi.fn(),
  isLeader: vi.fn(),
  isMember: vi.fn(),
};

vi.mock('@/stores', () => ({
  useAuthStore: () => mockAuthState,
}));

// Mock ForceChangePasswordModal
vi.mock('../ForceChangePasswordModal', () => ({
  default: () => <div data-testid="force-change-password">ForceChangePassword</div>,
}));

import AuthGuard from '../AuthGuard';

function renderGuard(
  roles?: ('admin' | 'leader' | 'member')[],
  initialEntries = ['/protected'],
) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <Routes>
        <Route
          path="/protected"
          element={
            <AuthGuard roles={roles}>
              <div>Protected Content</div>
            </AuthGuard>
          }
        />
        <Route path="/login" element={<div>Login Page</div>} />
        <Route path="/403" element={<div>Forbidden Page</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('AuthGuard', () => {
  beforeEach(() => {
    mockAuthState.isAuthenticated = false;
    mockAuthState.loading = false;
    mockAuthState.user = null;
  });

  it('loading 时应显示 Spin 加载指示器', () => {
    mockAuthState.loading = true;
    renderGuard();
    expect(document.querySelector('.ant-spin')).toBeTruthy();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('未认证时应跳转到 /login', () => {
    mockAuthState.isAuthenticated = false;
    mockAuthState.loading = false;
    renderGuard();
    expect(screen.getByText('Login Page')).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('已认证时应渲染子组件', () => {
    mockAuthState.isAuthenticated = true;
    mockAuthState.user = { id: '1', role: 'admin', must_change_password: false };
    renderGuard();
    expect(screen.getByText('Protected Content')).toBeInTheDocument();
  });

  it('角色不匹配时应跳转到 /403', () => {
    mockAuthState.isAuthenticated = true;
    mockAuthState.user = { id: '1', role: 'member', must_change_password: false };
    renderGuard(['admin']);
    expect(screen.getByText('Forbidden Page')).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('角色匹配时应正常渲染', () => {
    mockAuthState.isAuthenticated = true;
    mockAuthState.user = { id: '1', role: 'admin', must_change_password: false };
    renderGuard(['admin', 'leader']);
    expect(screen.getByText('Protected Content')).toBeInTheDocument();
  });

  it('must_change_password 时应显示强制改密弹窗', () => {
    mockAuthState.isAuthenticated = true;
    mockAuthState.user = { id: '1', role: 'admin', must_change_password: true };
    renderGuard();
    expect(screen.getByTestId('force-change-password')).toBeInTheDocument();
    expect(screen.getByText('Protected Content')).toBeInTheDocument();
  });
});
