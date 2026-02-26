import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Dropdown, Avatar, Button, theme } from 'antd';
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
  LogoutOutlined,
} from '@ant-design/icons';
import { useAuthStore, useAppStore } from '@/stores';
import { menuItems } from '@/constants/menus';

const { Header, Sider, Content } = Layout;

export default function AppLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { sidebarCollapsed, setSidebarCollapsed } = useAppStore();
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken();

  // 按角色过滤菜单
  const filteredMenus = menuItems.filter(
    (item) =>
      item.roles.length === 0 ||
      (user && item.roles.includes(user.role)),
  );

  const antdMenuItems = filteredMenus.map((item) => ({
    key: item.key,
    icon: item.icon,
    label: item.label,
    children: item.children
      ?.filter(
        (child) =>
          child.roles.length === 0 ||
          (user && child.roles.includes(user.role)),
      )
      .map((child) => ({
        key: child.key,
        label: child.label,
      })),
  }));

  // 根据 key 找到路径
  const findPath = (key: string): string | undefined => {
    for (const item of menuItems) {
      if (item.key === key) return item.path;
      if (item.children) {
        const child = item.children.find((c) => c.key === key);
        if (child) return child.path;
      }
    }
    return undefined;
  };

  const handleMenuClick = ({ key }: { key: string }) => {
    const path = findPath(key);
    if (path) navigate(path);
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const userMenuItems = [
    { key: 'profile', icon: <UserOutlined />, label: '个人中心' },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
  ];

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') handleLogout();
    else if (key === 'profile') navigate('/profile');
  };

  // 计算选中的菜单 key
  const selectedKeys = menuItems
    .flatMap((item) => [item, ...(item.children ?? [])])
    .filter((item) => location.pathname.startsWith(item.path))
    .sort((a, b) => b.path.length - a.path.length)
    .slice(0, 1)
    .map((item) => item.key);

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        trigger={null}
        collapsible
        collapsed={sidebarCollapsed}
        breakpoint="lg"
        onBreakpoint={(broken) => setSidebarCollapsed(broken)}
      >
        <div
          style={{
            height: 32,
            margin: 16,
            color: '#fff',
            textAlign: 'center',
            fontSize: sidebarCollapsed ? 14 : 18,
            fontWeight: 'bold',
            lineHeight: '32px',
          }}
        >
          {sidebarCollapsed ? 'EU' : 'Echo Union'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={selectedKeys}
          items={antdMenuItems}
          onClick={handleMenuClick}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 24px',
            background: colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Button
            type="text"
            icon={
              sidebarCollapsed ? (
                <MenuUnfoldOutlined />
              ) : (
                <MenuFoldOutlined />
              )
            }
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
          />
          <Dropdown
            menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
            placement="bottomRight"
          >
            <div
              style={{
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
              }}
            >
              <Avatar icon={<UserOutlined />} />
              <span>{user?.name}</span>
            </div>
          </Dropdown>
        </Header>
        <Content
          style={{
            margin: 24,
            padding: 24,
            background: colorBgContainer,
            borderRadius: borderRadiusLG,
            minHeight: 280,
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
