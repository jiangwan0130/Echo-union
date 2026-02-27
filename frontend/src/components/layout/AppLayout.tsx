import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Dropdown, Avatar, theme } from 'antd';
import { UserOutlined, LogoutOutlined } from '@ant-design/icons';
import { useAuthStore } from '@/stores';
import { menuItems, userDropdownItems } from '@/constants/menus';

const { Header, Content } = Layout;

export default function AppLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
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
  }));

  // 计算选中的菜单 key
  const selectedKeys = (() => {
    const pathname = location.pathname;
    // 精确匹配 /users 路由
    if (pathname.startsWith('/users')) return ['users'];
    // 其他都归到工作台
    return ['workbench'];
  })();

  const handleMenuClick = ({ key }: { key: string }) => {
    const item = menuItems.find((m) => m.key === key);
    if (item) navigate(item.path);
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') handleLogout();
    else if (key === 'profile') navigate('/profile');
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          padding: '0 24px',
          background: colorBgContainer,
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <div
          style={{
            fontSize: 18,
            fontWeight: 'bold',
            marginRight: 40,
            color: '#1677ff',
            cursor: 'pointer',
            whiteSpace: 'nowrap',
          }}
          onClick={() => navigate('/')}
        >
          Echo Union
        </div>
        <Menu
          mode="horizontal"
          selectedKeys={selectedKeys}
          items={antdMenuItems}
          onClick={handleMenuClick}
          style={{ flex: 1, border: 'none' }}
        />
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Dropdown
            menu={{ items: userDropdownItems, onClick: handleUserMenuClick }}
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
              <Avatar icon={<UserOutlined />} size="small" />
              <span>{user?.name}</span>
            </div>
          </Dropdown>
        </div>
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
  );
}
