import { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Button,
  Space,
  Dropdown,
  Drawer,
  Modal,
  Form,
  Input,
  Switch,
  Select,
  Checkbox,
  message,
  Typography,
  Flex,
  Badge,
} from 'antd';
import {
  PlusOutlined,
  MoreOutlined,
  EditOutlined,
  TeamOutlined,
  StopOutlined,
  CheckCircleOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { PageHeader, StatusTag } from '@/components/common';
import { departmentApi } from '@/services/departmentApi';
import { semesterApi } from '@/services/configApi';
import { showError } from '@/services/errorHandler';
import { useAuthStore } from '@/stores';
import type {
  DepartmentDetail,
  CreateDepartmentRequest,
  UpdateDepartmentRequest,
  DepartmentMember,
  SemesterInfo,
} from '@/types';

const { Text } = Typography;

export default function DepartmentsPage() {
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  // ── 部门列表 ──
  const [departments, setDepartments] = useState<DepartmentDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const [showInactive, setShowInactive] = useState(false);

  // ── 新建/编辑弹窗 ──
  const [formModalOpen, setFormModalOpen] = useState(false);
  const [editingDept, setEditingDept] = useState<DepartmentDetail | null>(null);
  const [form] = Form.useForm<CreateDepartmentRequest & { is_active?: boolean }>();
  const [formLoading, setFormLoading] = useState(false);

  // ── 成员管理抽屉 ──
  const [memberDrawerOpen, setMemberDrawerOpen] = useState(false);
  const [memberDept, setMemberDept] = useState<DepartmentDetail | null>(null);
  const [members, setMembers] = useState<DepartmentMember[]>([]);
  const [memberLoading, setMemberLoading] = useState(false);
  const [selectedMemberIds, setSelectedMemberIds] = useState<string[]>([]);
  const [saveLoading, setSaveLoading] = useState(false);

  // ── 学期选择(成员抽屉) ──
  const [semesters, setSemesters] = useState<SemesterInfo[]>([]);
  const [selectedSemesterId, setSelectedSemesterId] = useState<string>('');

  // ── 删除确认 ──
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletingDept, setDeletingDept] = useState<DepartmentDetail | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // ── 获取部门列表 ──
  const fetchDepartments = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await departmentApi.list({ include_inactive: showInactive || undefined });
      const raw = data.data;
      setDepartments(Array.isArray(raw) ? raw : (raw as unknown as { list: DepartmentDetail[] }).list ?? []);
    } catch {
      message.error('获取部门列表失败');
    } finally {
      setLoading(false);
    }
  }, [showInactive]);

  // ── 获取学期列表 ──
  const fetchSemesters = useCallback(async () => {
    try {
      const { data } = await semesterApi.list();
      const raw = data.data;
      const list = Array.isArray(raw) ? raw : (raw as unknown as { list: SemesterInfo[] }).list ?? [];
      setSemesters(list);
      const active = list.find((s) => s.is_active);
      if (active) setSelectedSemesterId(active.id);
      else if (list.length > 0) setSelectedSemesterId(list[0].id);
    } catch {
      /* 静默 */
    }
  }, []);

  useEffect(() => {
    fetchDepartments();
  }, [fetchDepartments]);

  useEffect(() => {
    fetchSemesters();
  }, [fetchSemesters]);

  // ── 新建/编辑 ──
  const openCreate = () => {
    setEditingDept(null);
    form.resetFields();
    setFormModalOpen(true);
  };

  const openEdit = (dept: DepartmentDetail) => {
    setEditingDept(dept);
    form.setFieldsValue({
      name: dept.name,
      description: dept.description,
      is_active: dept.is_active,
    });
    setFormModalOpen(true);
  };

  const handleFormSubmit = async () => {
    try {
      const values = await form.validateFields();
      setFormLoading(true);
      if (editingDept) {
        const updateData: UpdateDepartmentRequest = {
          name: values.name,
          description: values.description,
          is_active: values.is_active,
        };
        await departmentApi.update(editingDept.id, updateData);
        message.success('部门已更新');
      } else {
        await departmentApi.create({ name: values.name, description: values.description });
        message.success('部门已创建');
      }
      setFormModalOpen(false);
      fetchDepartments();
    } catch (err) {
      showError(err, editingDept ? '更新部门失败' : '创建部门失败');
    } finally {
      setFormLoading(false);
    }
  };

  // ── 删除部门 ──
  const openDeleteConfirm = (dept: DepartmentDetail) => {
    setDeletingDept(dept);
    setDeleteModalOpen(true);
  };

  const handleDelete = async () => {
    if (!deletingDept) return;
    setDeleteLoading(true);
    try {
      await departmentApi.delete(deletingDept.id);
      message.success('部门已删除');
      setDeleteModalOpen(false);
      setDeletingDept(null);
      fetchDepartments();
    } catch (err) {
      showError(err, '删除失败');
    } finally {
      setDeleteLoading(false);
    }
  };

  // ── 成员管理 ──
  const openMemberDrawer = async (dept: DepartmentDetail) => {
    setMemberDept(dept);
    setMemberDrawerOpen(true);
    await fetchMembers(dept.id, selectedSemesterId);
  };

  const fetchMembers = async (deptId: string, semesterId?: string) => {
    setMemberLoading(true);
    try {
      const { data } = await departmentApi.getMembers(deptId, semesterId || undefined);
      const raw = data.data;
      const list = Array.isArray(raw) ? raw : (raw as unknown as { list: DepartmentMember[] }).list ?? [];
      setMembers(list);
      setSelectedMemberIds(list.filter((m) => m.duty_required).map((m) => m.user_id));
    } catch {
      message.error('获取成员列表失败');
    } finally {
      setMemberLoading(false);
    }
  };

  const handleSemesterChange = (semId: string) => {
    setSelectedSemesterId(semId);
    if (memberDept) fetchMembers(memberDept.id, semId);
  };

  const handleToggleMember = (userId: string, checked: boolean) => {
    setSelectedMemberIds((prev) =>
      checked ? [...prev, userId] : prev.filter((id) => id !== userId),
    );
  };

  const handleSaveDutyMembers = async () => {
    if (!memberDept || !selectedSemesterId) return;
    setSaveLoading(true);
    try {
      await departmentApi.setDutyMembers(memberDept.id, {
        semester_id: selectedSemesterId,
        user_ids: selectedMemberIds,
      });
      message.success('值班人员已更新');
      setMemberDrawerOpen(false);
      fetchDepartments();
    } catch (err) {
      showError(err, '保存失败');
    } finally {
      setSaveLoading(false);
    }
  };

  // ── 表格列 ──
  const columns: ColumnsType<DepartmentDetail> = [
    {
      title: '部门名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => <Text strong>{name}</Text>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (desc: string | undefined) => desc ?? <Text type="secondary">—</Text>,
    },
    {
      title: '成员',
      dataIndex: 'member_count',
      key: 'member_count',
      width: 80,
      render: (count: number) => (
        <Badge count={count} showZero color={count > 0 ? 'blue' : 'default'} overflowCount={999} />
      ),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 80,
      render: (active: boolean) => <StatusTag active={active} />,
    },
    {
      title: '操作',
      key: 'actions',
      width: 160,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<TeamOutlined />}
            onClick={() => openMemberDrawer(record)}
          >
            管理成员
          </Button>
          {isAdmin && (
            <Dropdown
              menu={{
                items: [
                  { key: 'edit', icon: <EditOutlined />, label: '编辑' },
                  { type: 'divider' },
                  {
                    key: 'delete',
                    icon: <DeleteOutlined />,
                    label: '删除',
                    danger: true,
                  },
                ],
                onClick: ({ key }) => {
                  if (key === 'edit') openEdit(record);
                  else if (key === 'delete') openDeleteConfirm(record);
                },
              }}
              trigger={['click']}
            >
              <Button type="text" size="small" icon={<MoreOutlined />} />
            </Dropdown>
          )}
        </Space>
      ),
    },
  ];

  // ── 成员表格列 ──
  const memberColumns: ColumnsType<DepartmentMember> = [
    {
      title: (
        <Checkbox
          checked={members.length > 0 && selectedMemberIds.length === members.length}
          indeterminate={selectedMemberIds.length > 0 && selectedMemberIds.length < members.length}
          onChange={(e) =>
            setSelectedMemberIds(e.target.checked ? members.map((m) => m.user_id) : [])
          }
        />
      ),
      key: 'select',
      width: 48,
      render: (_, record) => (
        <Checkbox
          checked={selectedMemberIds.includes(record.user_id)}
          onChange={(e) => handleToggleMember(record.user_id, e.target.checked)}
        />
      ),
    },
    {
      title: '姓名',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '学号',
      dataIndex: 'student_id',
      key: 'student_id',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      ellipsis: true,
    },
    {
      title: '时间表',
      dataIndex: 'timetable_status',
      key: 'timetable_status',
      width: 100,
      render: (status: string) =>
        status === 'submitted' ? (
          <Badge status="success" text="已提交" />
        ) : (
          <Badge status="default" text="未提交" />
        ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="部门管理"
        description="管理各部门信息及成员值班配置"
        extra={
          <Space>
            <Flex align="center" gap={6}>
              <Text type="secondary" style={{ fontSize: 13 }}>显示停用</Text>
              <Switch size="small" checked={showInactive} onChange={setShowInactive} />
            </Flex>
            {isAdmin && (
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
                新建部门
              </Button>
            )}
          </Space>
        }
      />

      <Table<DepartmentDetail>
        rowKey="id"
        columns={columns}
        dataSource={departments}
        loading={loading}
        pagination={false}
      />

      {/* 新建/编辑弹窗 */}
      <Modal
        title={editingDept ? '编辑部门' : '新建部门'}
        open={formModalOpen}
        onCancel={() => setFormModalOpen(false)}
        onOk={handleFormSubmit}
        confirmLoading={formLoading}
        okText={editingDept ? '保存' : '创建'}
        cancelText="取消"
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label="部门名称"
            rules={[
              { required: true, message: '请输入部门名称' },
              { min: 2, max: 50, message: '名称长度 2-50 个字符' },
            ]}
          >
            <Input placeholder="请输入部门名称" />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
            rules={[{ max: 200, message: '描述最多 200 个字符' }]}
          >
            <Input.TextArea placeholder="请输入描述（可选）" rows={3} showCount maxLength={200} />
          </Form.Item>
          {editingDept && (
            <Form.Item name="is_active" label="启用状态" valuePropName="checked">
              <Switch checkedChildren={<CheckCircleOutlined />} unCheckedChildren={<StopOutlined />} />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* 成员管理抽屉 */}
      <Drawer
        title={
          <Flex align="center" gap={8}>
            <TeamOutlined />
            <span>{memberDept?.name} — 成员管理</span>
          </Flex>
        }
        open={memberDrawerOpen}
        onClose={() => setMemberDrawerOpen(false)}
        width={640}
        extra={
          <Space>
            <Button onClick={() => setMemberDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={saveLoading} onClick={handleSaveDutyMembers}>
              保存值班人员
            </Button>
          </Space>
        }
      >
        <Flex gap={12} align="center" style={{ marginBottom: 16 }}>
          <Text type="secondary">选择学期：</Text>
          <Select
            style={{ width: 240 }}
            value={selectedSemesterId || undefined}
            onChange={handleSemesterChange}
            options={semesters.map((s) => ({
              label: s.name + (s.is_active ? ' (当前)' : ''),
              value: s.id,
            }))}
            placeholder="请选择学期"
          />
        </Flex>
        <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
          ☑ 勾选的成员将被标记为需要值班，排班时将自动纳入。
        </Text>
        <Table<DepartmentMember>
          rowKey="user_id"
          columns={memberColumns}
          dataSource={members}
          loading={memberLoading}
          pagination={false}
          size="small"
        />
        {members.length > 0 && (
          <Text type="secondary" style={{ display: 'block', marginTop: 12 }}>
            已选 {selectedMemberIds.length} / {members.length} 名成员
          </Text>
        )}
      </Drawer>

      {/* 删除确认弹窗 */}
      <Modal
        title="确认删除"
        open={deleteModalOpen}
        onCancel={() => { setDeleteModalOpen(false); setDeletingDept(null); }}
        onOk={handleDelete}
        confirmLoading={deleteLoading}
        okText="删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <p>确定要删除部门 <Text strong>{deletingDept?.name}</Text> 吗？</p>
        <p><Text type="secondary">删除前请确保该部门下没有成员，否则无法删除。</Text></p>
      </Modal>
    </div>
  );
}
