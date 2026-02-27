import {
  DashboardOutlined,
  TeamOutlined,
  UserOutlined,
  LogoutOutlined,
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

// 顶部导航菜单项（精简版：工作台 + 用户管理）
export const menuItems: MenuItem[] = [
  {
    key: 'workbench',
    label: '工作台',
    icon: <DashboardOutlined />,
    path: '/',
    roles: [],
  },
  {
    key: 'users',
    label: '用户管理',
    icon: <TeamOutlined />,
    path: '/users',
    roles: ['admin'],
  },
];

// 头像下拉菜单项
export const userDropdownItems = [
  { key: 'profile', icon: <UserOutlined />, label: '个人中心' },
  { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
];
