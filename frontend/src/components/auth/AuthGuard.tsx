import { Navigate, useLocation } from 'react-router-dom';
import { Spin } from 'antd';
import { useAuthStore } from '@/stores';
import type { UserRole } from '@/types';
import ForceChangePasswordModal from './ForceChangePasswordModal';

interface AuthGuardProps {
  children: React.ReactNode;
  roles?: UserRole[];
}

export default function AuthGuard({ children, roles }: AuthGuardProps) {
  const { isAuthenticated, loading, user } = useAuthStore();
  const location = useLocation();

  if (loading) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '100vh',
        }}
      >
        <Spin size="large" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (roles && roles.length > 0 && user && !roles.includes(user.role)) {
    return <Navigate to="/403" replace />;
  }

  return (
    <>
      {user?.must_change_password && <ForceChangePasswordModal />}
      {children}
    </>
  );
}
