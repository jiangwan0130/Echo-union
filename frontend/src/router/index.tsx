import { createBrowserRouter, Navigate } from 'react-router-dom';
import { lazy, Suspense } from 'react';
import { Spin } from 'antd';
import AppLayout from '@/components/layout/AppLayout';
import AuthGuard from '@/components/auth/AuthGuard';

// ── 懒加载页面 ──
const LoginPage = lazy(() => import('@/pages/login/LoginPage'));
const DashboardPage = lazy(() => import('@/pages/dashboard/DashboardPage'));
const TimetablePage = lazy(() => import('@/pages/timetable/TimetablePage'));
const SchedulePage = lazy(() => import('@/pages/schedule/SchedulePage'));
const AutoSchedulePage = lazy(
  () => import('@/pages/schedule/AutoSchedulePage'),
);
const AdjustSchedulePage = lazy(
  () => import('@/pages/schedule/AdjustSchedulePage'),
);
const UsersPage = lazy(() => import('@/pages/admin/users/UsersPage'));
const DepartmentsPage = lazy(
  () => import('@/pages/admin/departments/DepartmentsPage'),
);
const ConfigPage = lazy(() => import('@/pages/admin/config/ConfigPage'));
const ProgressPage = lazy(
  () => import('@/pages/admin/progress/ProgressPage'),
);
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
      { index: true, element: <Navigate to="/dashboard" replace /> },
      {
        path: 'dashboard',
        element: (
          <LazyLoad>
            <DashboardPage />
          </LazyLoad>
        ),
      },
      {
        path: 'schedule',
        element: (
          <LazyLoad>
            <SchedulePage />
          </LazyLoad>
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
      {
        path: 'timetable',
        element: (
          <AuthGuard roles={['member']}>
            <LazyLoad>
              <TimetablePage />
            </LazyLoad>
          </AuthGuard>
        ),
      },
      {
        path: 'schedule/auto',
        element: (
          <AuthGuard roles={['admin']}>
            <LazyLoad>
              <AutoSchedulePage />
            </LazyLoad>
          </AuthGuard>
        ),
      },
      {
        path: 'schedule/adjust',
        element: (
          <AuthGuard roles={['admin']}>
            <LazyLoad>
              <AdjustSchedulePage />
            </LazyLoad>
          </AuthGuard>
        ),
      },
      {
        path: 'admin/users',
        element: (
          <AuthGuard roles={['admin']}>
            <LazyLoad>
              <UsersPage />
            </LazyLoad>
          </AuthGuard>
        ),
      },
      {
        path: 'admin/departments',
        element: (
          <AuthGuard roles={['admin', 'leader']}>
            <LazyLoad>
              <DepartmentsPage />
            </LazyLoad>
          </AuthGuard>
        ),
      },
      {
        path: 'admin/config',
        element: (
          <AuthGuard roles={['admin']}>
            <LazyLoad>
              <ConfigPage />
            </LazyLoad>
          </AuthGuard>
        ),
      },
      {
        path: 'admin/progress',
        element: (
          <AuthGuard roles={['admin', 'leader']}>
            <LazyLoad>
              <ProgressPage />
            </LazyLoad>
          </AuthGuard>
        ),
      },
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
