import { Typography, Flex } from 'antd';
import { useAuthStore } from '@/stores';
import AdminDashboard from './AdminDashboard';
import LeaderDashboard from './LeaderDashboard';
import MemberDashboard from './MemberDashboard';

const { Title, Text } = Typography;

export default function DashboardPage() {
  const { user } = useAuthStore();
  const role = user?.role;

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

      {role === 'admin' && <AdminDashboard />}
      {role === 'leader' && <LeaderDashboard />}
      {role === 'member' && <MemberDashboard />}
    </div>
  );
}
