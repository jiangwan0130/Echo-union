import {
  DashboardOutlined,
  CalendarOutlined,
  ScheduleOutlined,
  TeamOutlined,
  ApartmentOutlined,
  SettingOutlined,
  UserOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import type { ReactNode } from 'react';
import type { UserRole } from '@/types';

export interface MenuItem {
  key: string;
  label: string;
  icon: ReactNode;
  path: string;
  roles: UserRole[]; // 空数组 = 所有角色可见
  children?: MenuItem[];
}

export const menuItems: MenuItem[] = [
  {
    key: 'dashboard',
    label: '工作台',
    icon: <DashboardOutlined />,
    path: '/dashboard',
    roles: [],
  },
  {
    key: 'timetable',
    label: '我的时间表',
    icon: <CalendarOutlined />,
    path: '/timetable',
    roles: ['member'],
  },
  {
    key: 'schedule',
    label: '排班概览',
    icon: <ScheduleOutlined />,
    path: '/schedule',
    roles: [],
  },
  {
    key: 'schedule-manage',
    label: '排班管理',
    icon: <ScheduleOutlined />,
    path: '/schedule/auto',
    roles: ['admin'],
    children: [
      {
        key: 'schedule-auto',
        label: '自动排班',
        icon: <ScheduleOutlined />,
        path: '/schedule/auto',
        roles: ['admin'],
      },
      {
        key: 'schedule-adjust',
        label: '手动调整',
        icon: <ScheduleOutlined />,
        path: '/schedule/adjust',
        roles: ['admin'],
      },
    ],
  },
  {
    key: 'admin-users',
    label: '用户管理',
    icon: <TeamOutlined />,
    path: '/admin/users',
    roles: ['admin'],
  },
  {
    key: 'admin-departments',
    label: '部门管理',
    icon: <ApartmentOutlined />,
    path: '/admin/departments',
    roles: ['admin', 'leader'],
  },
  {
    key: 'admin-progress',
    label: '提交进度',
    icon: <BarChartOutlined />,
    path: '/admin/progress',
    roles: ['admin', 'leader'],
  },
  {
    key: 'admin-config',
    label: '系统配置',
    icon: <SettingOutlined />,
    path: '/admin/config',
    roles: ['admin'],
  },
  {
    key: 'profile',
    label: '个人中心',
    icon: <UserOutlined />,
    path: '/profile',
    roles: [],
  },
];
