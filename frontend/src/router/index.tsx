import { createBrowserRouter, Navigate } from 'react-router-dom';
import { lazy, Suspense } from 'react';
import { Spin } from 'antd';
import AppLayout from '@/components/layout/AppLayout';
import AuthGuard from '@/components/auth/AuthGuard';

// ── 懒加载页面 ──
const LoginPage = lazy(() => import('@/pages/login/LoginPage'));
const WorkbenchPage = lazy(() => import('@/pages/workbench/WorkbenchPage'));
const UsersPage = lazy(() => import('@/pages/admin/users/UsersPage'));
const ProfilePage = lazy(() => import('@/pages/profile/ProfilePage'));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));
const ForbiddenPage = lazy(() => import('@/pages/ForbiddenPage'));

function LazyLoad({ children }: { children: React.ReactNode }) {
  return (
    <Suspense
      fallback={
        <div
          style={{
            display: 'flex',
            justifyContent: 'center',
            padding: 100,
          }}
        >
          <Spin size="large" />
        </div>
      }
    >
      {children}
    </Suspense>
  );
}

export const router = createBrowserRouter([
  {
    path: '/login',
    element: (
      <LazyLoad>
        <LoginPage />
      </LazyLoad>
    ),
  },
  {
    path: '/',
    element: (
      <AuthGuard>
        <AppLayout />
      </AuthGuard>
    ),
    children: [
      {
        index: true,
        element: (
          <LazyLoad>
            <WorkbenchPage />
          </LazyLoad>
        ),
      },
      {
        path: 'users',
        element: (
          <AuthGuard roles={['admin']}>
            <LazyLoad>
              <UsersPage />
            </LazyLoad>
          </AuthGuard>
        ),
      },
      {
        path: 'profile',
        element: (
          <LazyLoad>
            <ProfilePage />
          </LazyLoad>
        ),
      },
      // ── 旧路由重定向到工作台 ──
      { path: 'dashboard', element: <Navigate to="/" replace /> },
      { path: 'schedule', element: <Navigate to="/" replace /> },
      { path: 'schedule/auto', element: <Navigate to="/" replace /> },
      { path: 'schedule/adjust', element: <Navigate to="/" replace /> },
      { path: 'timetable', element: <Navigate to="/" replace /> },
      { path: 'admin/config', element: <Navigate to="/" replace /> },
      { path: 'admin/departments', element: <Navigate to="/users" replace /> },
      { path: 'admin/progress', element: <Navigate to="/" replace /> },
      { path: 'admin/users', element: <Navigate to="/users" replace /> },
    ],
  },
  {
    path: '/403',
    element: (
      <LazyLoad>
        <ForbiddenPage />
      </LazyLoad>
    ),
  },
  {
    path: '*',
    element: (
      <LazyLoad>
        <NotFoundPage />
      </LazyLoad>
    ),
  },
]);
