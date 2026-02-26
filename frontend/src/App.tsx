import { useEffect } from 'react';
import { RouterProvider } from 'react-router-dom';
import { ConfigProvider, App as AntdApp } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { router } from '@/router';
import { useAuthStore, useAppStore } from '@/stores';

export default function App() {
  const fetchCurrentUser = useAuthStore((s) => s.fetchCurrentUser);
  const fetchCurrentSemester = useAppStore((s) => s.fetchCurrentSemester);

  useEffect(() => {
    fetchCurrentUser().then(() => {
      fetchCurrentSemester();
    });
  }, [fetchCurrentUser, fetchCurrentSemester]);

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#1677ff',
          borderRadius: 6,
        },
      }}
    >
      <AntdApp>
        <RouterProvider router={router} />
      </AntdApp>
    </ConfigProvider>
  );
}
