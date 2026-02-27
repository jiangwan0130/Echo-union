import { Typography, Flex } from 'antd';
import { useAuthStore, useAppStore } from '@/stores';
import { useEffect } from 'react';
import AdminWorkbench from './AdminWorkbench';
import LeaderWorkbench from './LeaderWorkbench';
import MemberWorkbench from './MemberWorkbench';

const { Title, Text } = Typography;

export default function WorkbenchPage() {
  const { user } = useAuthStore();
  const fetchPendingTodos = useAppStore((s) => s.fetchPendingTodos);
  const fetchCurrentSemester = useAppStore((s) => s.fetchCurrentSemester);
  const role = user?.role;

  useEffect(() => {
    fetchCurrentSemester();
    fetchPendingTodos();
  }, [fetchCurrentSemester, fetchPendingTodos]);

  return (
    <div>
      <Flex justify="space-between" align="flex-start" style={{ marginBottom: 24 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            欢迎回来，{user?.name}
          </Title>
          <Text type="secondary">
            角色：{role === 'admin' ? '管理员' : role === 'leader' ? '负责人' : '成员'}
            {user?.department?.name ? ` | 部门：${user.department.name}` : ''}
          </Text>
        </div>
      </Flex>

      {role === 'admin' && <AdminWorkbench />}
      {role === 'leader' && <LeaderWorkbench />}
      {role === 'member' && <MemberWorkbench />}
    </div>
  );
}
