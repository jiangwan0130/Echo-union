import { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Input,
  Select,
  Button,
  Space,
  Tag,
  Dropdown,
  Drawer,
  Modal,
  Form,
  Radio,
  Upload,
  message,
  Typography,
  Alert,
  Flex,
} from 'antd';
import {
  SearchOutlined,
  UploadOutlined,
  MoreOutlined,
  EditOutlined,
  UserSwitchOutlined,
  KeyOutlined,
  DeleteOutlined,
  InboxOutlined,
  CopyOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { PageHeader, ConfirmAction } from '@/components/common';
import { userApi } from '@/services/userApi';
import { departmentApi } from '@/services/departmentApi';
import { showError } from '@/services/errorHandler';
import type {
  UserInfo,
  UserRole,
  UserListParams,
  UpdateUserRequest,
  DepartmentDetail,
  ImportUserResponse,
  ImportUserError,
} from '@/types';

const { Text } = Typography;
const { Dragger } = Upload;

const ROLE_MAP: Record<UserRole, { label: string; color: string }> = {
  admin: { label: '管理员', color: 'red' },
  leader: { label: '负责人', color: 'blue' },
  member: { label: '成员', color: 'default' },
};

const PAGE_SIZE = 20;

export default function UsersPage() {
  // ── 列表数据 ──
  const [users, setUsers] = useState<UserInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [params, setParams] = useState<UserListParams>({
    page: 1,
    page_size: PAGE_SIZE,
  });

  // ── 部门列表(筛选用) ──
  const [departments, setDepartments] = useState<DepartmentDetail[]>([]);

  // ── 编辑抽屉 ──
  const [editDrawerOpen, setEditDrawerOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<UserInfo | null>(null);
  const [editForm] = Form.useForm<UpdateUserRequest>();
  const [editLoading, setEditLoading] = useState(false);

  // ── 角色分配弹窗 ──
  const [roleModalOpen, setRoleModalOpen] = useState(false);
  const [roleUser, setRoleUser] = useState<UserInfo | null>(null);
  const [roleForm] = Form.useForm<{ role: UserRole }>();
  const [roleLoading, setRoleLoading] = useState(false);

  // ── 重置密码结果 ──
  const [tempPassword, setTempPassword] = useState('');
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);

  // ── 导入弹窗 ──
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [importLoading, setImportLoading] = useState(false);
  const [importResult, setImportResult] = useState<ImportUserResponse | null>(null);

  // ── 获取用户列表 ──
  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await userApi.listUsers(params);
      setUsers(data.data.list);
      setTotal(data.data.pagination.total);
    } catch {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  }, [params]);

  // ── 获取部门列表 ──
  const fetchDepartments = useCallback(async () => {
    try {
      const { data } = await departmentApi.list({ include_inactive: false });
      const raw = data.data;
      setDepartments(Array.isArray(raw) ? raw : (raw as unknown as { list: DepartmentDetail[] }).list ?? []);
    } catch {
      /* 静默 */
    }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  useEffect(() => {
    fetchDepartments();
  }, [fetchDepartments]);

  // ── 搜索/筛选 ──
  const handleSearch = (keyword: string) => {
    setParams((prev) => ({ ...prev, keyword: keyword || undefined, page: 1 }));
  };

  const handleDeptFilter = (deptId: string | undefined) => {
    setParams((prev) => ({ ...prev, department_id: deptId, page: 1 }));
  };

  const handleRoleFilter = (role: UserRole | undefined) => {
    setParams((prev) => ({ ...prev, role, page: 1 }));
  };

  const handleReset = () => {
    setParams({ page: 1, page_size: PAGE_SIZE });
  };

  // ── 删除确认弹窗 ──
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletingUser, setDeletingUser] = useState<UserInfo | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // ── 编辑用户 ──
  const openEdit = (user: UserInfo) => {
    setEditingUser(user);
    editForm.setFieldsValue({
      name: user.name,
      email: user.email,
      department_id: user.department?.id,
    });
    setEditDrawerOpen(true);
  };

  const handleEditSubmit = async () => {
    try {
      const values = await editForm.validateFields();
      setEditLoading(true);
      await userApi.updateUser(editingUser!.id, values);
      message.success('用户信息已更新');
      setEditDrawerOpen(false);
      fetchUsers();
    } catch (err) {
      showError(err, '更新用户信息失败');
    } finally {
      setEditLoading(false);
    }
  };

  // ── 角色分配 ──
  const openRoleModal = (user: UserInfo) => {
    setRoleUser(user);
    roleForm.setFieldsValue({ role: user.role });
    setRoleModalOpen(true);
  };

  const handleRoleSubmit = async () => {
    try {
      const { role } = await roleForm.validateFields();
      setRoleLoading(true);
      await userApi.assignRole(roleUser!.id, { role });
      message.success('角色已更新');
      setRoleModalOpen(false);
      fetchUsers();
    } catch (err) {
      showError(err, '分配角色失败');
    } finally {
      setRoleLoading(false);
    }
  };

  // ── 重置密码 ──
  const handleResetPassword = async (userId: string) => {
    try {
      const { data } = await userApi.resetPassword(userId);
      setTempPassword(data.data.temp_password);
      setPasswordModalOpen(true);
    } catch {
      message.error('重置密码失败');
    }
  };

  // ── 删除用户 ──
  const openDeleteConfirm = (user: UserInfo) => {
    setDeletingUser(user);
    setDeleteModalOpen(true);
  };

  const handleDelete = async () => {
    if (!deletingUser) return;
    setDeleteLoading(true);
    try {
      await userApi.deleteUser(deletingUser.id);
      message.success('用户已删除');
      setDeleteModalOpen(false);
      setDeletingUser(null);
      fetchUsers();
    } catch (err) {
      showError(err, '删除失败');
    } finally {
      setDeleteLoading(false);
    }
  };

  // ── 导入 ──
  const handleImport = async (file: File) => {
    setImportLoading(true);
    setImportResult(null);
    try {
      const { data } = await userApi.importUsers(file);
      setImportResult(data.data);
      if (data.data.failed === 0) {
        message.success(`成功导入 ${data.data.success} 名用户`);
      } else {
        message.warning(`导入完成，成功 ${data.data.success} 条，失败 ${data.data.failed} 条`);
      }
      fetchUsers();
    } catch {
      message.error('导入失败');
    } finally {
      setImportLoading(false);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    message.success('已复制到剪贴板');
  };

  // ── 表格列配置 ──
  const columns: ColumnsType<UserInfo> = [
    {
      title: '用户',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => <Text strong>{name}</Text>,
    },
    {
      title: '学号',
      dataIndex: 'student_id',
      key: 'student_id',
      render: (id: string) => <Text copyable={{ tooltips: ['复制', '已复制'] }}>{id}</Text>,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      ellipsis: true,
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      width: 100,
      render: (role: UserRole) => (
        <Tag color={ROLE_MAP[role].color} bordered={false}>
          {ROLE_MAP[role].label}
        </Tag>
      ),
    },
    {
      title: '部门',
      dataIndex: ['department', 'name'],
      key: 'department',
      render: (name: string | undefined) => name ?? <Text type="secondary">未分配</Text>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 60,
      render: (_, record) => (
        <Dropdown
          menu={{
            items: [
              { key: 'edit', icon: <EditOutlined />, label: '编辑信息' },
              { key: 'role', icon: <UserSwitchOutlined />, label: '分配角色' },
              { key: 'reset', icon: <KeyOutlined />, label: '重置密码' },
              { type: 'divider' },
              { key: 'delete', icon: <DeleteOutlined />, label: '删除用户', danger: true },
            ],
            onClick: ({ key }) => {
              if (key === 'edit') openEdit(record);
              else if (key === 'role') openRoleModal(record);
              else if (key === 'reset') handleResetPassword(record.id);
              else if (key === 'delete') openDeleteConfirm(record);
            },
          }}
          trigger={['click']}
        >
          <Button type="text" icon={<MoreOutlined />} />
        </Dropdown>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="用户管理"
        description="管理系统中的所有用户账号、角色与部门归属"
        extra={
          <Button icon={<UploadOutlined />} onClick={() => { setImportResult(null); setImportModalOpen(true); }}>
            批量导入
          </Button>
        }
      />

      {/* 搜索工具栏 */}
      <Flex gap={12} wrap="wrap" style={{ marginBottom: 16 }}>
        <Input
          placeholder="搜索姓名 / 学号"
          prefix={<SearchOutlined />}
          allowClear
          style={{ width: 220 }}
          value={params.keyword}
          onChange={(e) => handleSearch(e.target.value)}
        />
        <Select
          placeholder="部门"
          allowClear
          style={{ width: 160 }}
          value={params.department_id}
          onChange={handleDeptFilter}
          options={departments.map((d) => ({ label: d.name, value: d.id }))}
        />
        <Select
          placeholder="角色"
          allowClear
          style={{ width: 120 }}
          value={params.role}
          onChange={handleRoleFilter}
          options={[
            { label: '管理员', value: 'admin' },
            { label: '负责人', value: 'leader' },
            { label: '成员', value: 'member' },
          ]}
        />
        <Button onClick={handleReset}>重置</Button>
      </Flex>

      {/* 用户表格 */}
      <Table<UserInfo>
        rowKey="id"
        columns={columns}
        dataSource={users}
        loading={loading}
        pagination={{
          current: params.page,
          pageSize: params.page_size,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (page, pageSize) =>
            setParams((prev) => ({ ...prev, page, page_size: pageSize })),
        }}
      />

      {/* 编辑抽屉 */}
      <Drawer
        title="编辑用户信息"
        open={editDrawerOpen}
        onClose={() => setEditDrawerOpen(false)}
        width={400}
        extra={
          <Space>
            <Button onClick={() => setEditDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={editLoading} onClick={handleEditSubmit}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="name"
            label="姓名"
            rules={[
              { required: true, message: '请输入姓名' },
              { min: 2, max: 20, message: '姓名长度 2-20 个字符' },
            ]}
          >
            <Input placeholder="请输入姓名" />
          </Form.Item>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>
          <Form.Item name="department_id" label="所属部门">
            <Select
              placeholder="请选择部门"
              allowClear
              options={departments.map((d) => ({ label: d.name, value: d.id }))}
            />
          </Form.Item>
        </Form>
      </Drawer>

      {/* 角色分配弹窗 */}
      <Modal
        title={`分配角色 — ${roleUser?.name ?? ''}`}
        open={roleModalOpen}
        onCancel={() => setRoleModalOpen(false)}
        onOk={handleRoleSubmit}
        confirmLoading={roleLoading}
        okText="确认"
        cancelText="取消"
      >
        <Form form={roleForm} style={{ marginTop: 16 }}>
          <Form.Item name="role" label="角色">
            <Radio.Group>
              <Radio.Button value="admin">管理员</Radio.Button>
              <Radio.Button value="leader">负责人</Radio.Button>
              <Radio.Button value="member">成员</Radio.Button>
            </Radio.Group>
          </Form.Item>
        </Form>
      </Modal>

      {/* 重置密码结果弹窗 */}
      <Modal
        title="密码已重置"
        open={passwordModalOpen}
        onCancel={() => setPasswordModalOpen(false)}
        footer={
          <Button type="primary" onClick={() => setPasswordModalOpen(false)}>
            知道了
          </Button>
        }
      >
        <Alert
          type="success"
          showIcon
          message="临时密码"
          description={
            <Flex align="center" gap={8}>
              <Text code style={{ fontSize: 16 }}>
                {tempPassword}
              </Text>
              <Button
                type="link"
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(tempPassword)}
              >
                复制
              </Button>
            </Flex>
          }
          style={{ marginTop: 8 }}
        />
        <Text type="secondary" style={{ display: 'block', marginTop: 12 }}>
          请将临时密码发送给用户，用户登录后可自行修改密码。
        </Text>
      </Modal>

      {/* 删除确认弹窗 */}
      <Modal
        title="确认删除"
        open={deleteModalOpen}
        onCancel={() => { setDeleteModalOpen(false); setDeletingUser(null); }}
        onOk={handleDelete}
        confirmLoading={deleteLoading}
        okText="删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <p>确定要删除用户 <Text strong>{deletingUser?.name}</Text>（学号：{deletingUser?.student_id}）吗？</p>
        <p><Text type="secondary">此操作不可撤销，请谨慎操作。</Text></p>
      </Modal>

      {/* 批量导入弹窗 */}
      <Modal
        title="批量导入用户"
        open={importModalOpen}
        onCancel={() => setImportModalOpen(false)}
        footer={importResult ? (
          <Button type="primary" onClick={() => setImportModalOpen(false)}>
            完成
          </Button>
        ) : null}
        width={520}
      >
        {!importResult ? (
          <>
            <Alert
              type="info"
              showIcon
              message="导入说明"
              description="请上传 .xlsx 格式的 Excel 文件，文件大小不超过 5MB。文件需包含：姓名、学号、邮箱等列。"
              style={{ marginBottom: 16 }}
            />
            <Dragger
              accept=".xlsx"
              maxCount={1}
              showUploadList={false}
              beforeUpload={(file) => {
                if (file.size > 5 * 1024 * 1024) {
                  message.error('文件大小不能超过 5MB');
                  return false;
                }
                handleImport(file);
                return false;
              }}
              disabled={importLoading}
            >
              <p className="ant-upload-drag-icon">
                <InboxOutlined />
              </p>
              <p className="ant-upload-text">
                {importLoading ? '正在导入...' : '点击或拖拽文件到此区域'}
              </p>
              <p className="ant-upload-hint">仅支持 .xlsx 格式</p>
            </Dragger>
          </>
        ) : (
          <div>
            <Alert
              type={importResult.failed === 0 ? 'success' : 'warning'}
              showIcon
              message={`导入完成：总计 ${importResult.total} 条，成功 ${importResult.success} 条，失败 ${importResult.failed} 条`}
              style={{ marginBottom: 16 }}
            />
            {importResult.errors && importResult.errors.length > 0 && (
              <Table<ImportUserError>
                rowKey="row"
                size="small"
                dataSource={importResult.errors}
                columns={[
                  { title: '行号', dataIndex: 'row', key: 'row', width: 80 },
                  { title: '原因', dataIndex: 'reason', key: 'reason' },
                ]}
                pagination={false}
                scroll={{ y: 200 }}
              />
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
