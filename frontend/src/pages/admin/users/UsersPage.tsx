import { Tabs } from 'antd';
import { TeamOutlined, ApartmentOutlined, CalendarOutlined } from '@ant-design/icons';
import { PageHeader } from '@/components/common';
import UserListTab from './UserListTab';
import DepartmentsPage from '@/pages/admin/departments/DepartmentsPage';
import SemesterTab from '@/pages/admin/config/SemesterTab';

export default function UsersPage() {
  return (
    <div>
      <PageHeader
        title="用户管理"
        description="管理系统成员、部门与学期"
      />
      <Tabs
        defaultActiveKey="users"
        items={[
          {
            key: 'users',
            label: (
              <span>
                <TeamOutlined style={{ marginRight: 6 }} />
                成员列表
              </span>
            ),
            children: <UserListTab />,
          },
          {
            key: 'departments',
            label: (
              <span>
                <ApartmentOutlined style={{ marginRight: 6 }} />
                部门管理
              </span>
            ),
            children: <DepartmentsPage />,
          },
          {
            key: 'semesters',
            label: (
              <span>
                <CalendarOutlined style={{ marginRight: 6 }} />
                学期管理
              </span>
            ),
            children: <SemesterTab />,
          },
        ]}
      />
    </div>
  );
}
