import { useState, useEffect, useCallback } from 'react';
import {
  Tabs,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  DatePicker,
  Radio,
  Switch,
  Tag,
  Tooltip,
  Spin,
  message,
  Typography,
  Flex,
  Empty,
} from 'antd';
import {
  PlusOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
  EnvironmentOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  ThunderboltOutlined,
  StarFilled,
  StarOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader, StatusTag, ConfirmAction } from '@/components/common';
import WeekGridView from '@/components/config/WeekGridView';
import { semesterApi, timeSlotApi, locationApi, scheduleRuleApi, systemConfigApi } from '@/services/configApi';
import { showError } from '@/services/errorHandler';
import type {
  SemesterInfo,
  CreateSemesterRequest,
  UpdateSemesterRequest,
  TimeSlotInfo,
  LocationInfo,
  CreateLocationRequest,
  UpdateLocationRequest,
  ScheduleRuleInfo,
  SystemConfigInfo,
  UpdateSystemConfigRequest,
} from '@/types';

const { Text } = Typography;
const { RangePicker } = DatePicker;

// ════════════════════════════════════════
// Tab 1: 学期管理
// ════════════════════════════════════════
function SemesterTab() {
  const [semesters, setSemesters] = useState<SemesterInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<SemesterInfo | null>(null);
  const [form] = Form.useForm();
  const [submitLoading, setSubmitLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await semesterApi.list();
      const raw = data.data;
      setSemesters(Array.isArray(raw) ? raw : (raw as unknown as { list: SemesterInfo[] }).list ?? []);
    } catch {
      message.error('获取学期列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (item: SemesterInfo) => {
    setEditing(item);
    form.setFieldsValue({
      name: item.name,
      date_range: [dayjs(item.start_date), dayjs(item.end_date)],
      first_week_type: item.first_week_type,
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitLoading(true);
      const [start, end] = values.date_range;

      if (editing) {
        const updateData: UpdateSemesterRequest = {
          name: values.name,
          start_date: start.format('YYYY-MM-DD'),
          end_date: end.format('YYYY-MM-DD'),
          first_week_type: values.first_week_type,
        };
        await semesterApi.update(editing.id, updateData);
        message.success('学期已更新');
      } else {
        const createData: CreateSemesterRequest = {
          name: values.name,
          start_date: start.format('YYYY-MM-DD'),
          end_date: end.format('YYYY-MM-DD'),
          first_week_type: values.first_week_type,
        };
        await semesterApi.create(createData);
        message.success('学期已创建');
      }
      setModalOpen(false);
      fetchData();
    } catch (err) {
      showError(err, editing ? '更新学期失败' : '创建学期失败');
    } finally {
      setSubmitLoading(false);
    }
  };

  const handleActivate = async (id: string) => {
    try {
      await semesterApi.activate(id);
      message.success('学期已激活');
      fetchData();
    } catch (err) {
      showError(err, '激活失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await semesterApi.delete(id);
      message.success('学期已删除');
      fetchData();
    } catch (err) {
      showError(err, '删除失败');
    }
  };

  const columns: ColumnsType<SemesterInfo> = [
    {
      title: '学期名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record) => (
        <Space>
          <Text strong>{name}</Text>
          {record.is_active && <Tag color="orange" bordered={false}>当前</Tag>}
        </Space>
      ),
    },
    {
      title: '日期范围',
      key: 'dates',
      render: (_, r) => (
        <Text>
          {dayjs(r.start_date).format('YYYY/MM/DD')} — {dayjs(r.end_date).format('YYYY/MM/DD')}
        </Text>
      ),
    },
    {
      title: '首周',
      dataIndex: 'first_week_type',
      key: 'first_week_type',
      width: 80,
      render: (type: string) => (
        <Tag color={type === 'odd' ? 'cyan' : 'purple'} bordered={false}>
          {type === 'odd' ? '单周' : '双周'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => {
        const map: Record<string, { color: string; label: string }> = {
          active: { color: 'green', label: '进行中' },
          upcoming: { color: 'blue', label: '未开始' },
          archived: { color: 'default', label: '已归档' },
        };
        const info = map[status] ?? { color: 'default', label: status };
        return <Tag color={info.color} bordered={false}>{info.label}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 180,
      render: (_, record) => (
        <Space>
          {!record.is_active && (
            <ConfirmAction
              title="激活此学期？"
              description="激活后，当前学期将切换为此学期。"
              onConfirm={() => handleActivate(record.id)}
              danger={false}
            >
              <Button type="link" size="small" icon={<ThunderboltOutlined />}>
                激活
              </Button>
            </ConfirmAction>
          )}
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>
            编辑
          </Button>
          {!record.is_active && (
            <ConfirmAction
              title="确认删除此学期？"
              description="删除后不可恢复。"
              onConfirm={() => handleDelete(record.id)}
            >
              <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                删除
              </Button>
            </ConfirmAction>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Flex justify="flex-end" style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建学期
        </Button>
      </Flex>
      <Table<SemesterInfo>
        rowKey="id"
        columns={columns}
        dataSource={semesters}
        loading={loading}
        pagination={false}
      />
      <Modal
        title={editing ? '编辑学期' : '新建学期'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmit}
        confirmLoading={submitLoading}
        okText={editing ? '保存' : '创建'}
        cancelText="取消"
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label="学期名称"
            rules={[
              { required: true, message: '请输入学期名称' },
              { min: 2, max: 100, message: '名称长度 2-100 个字符' },
            ]}
          >
            <Input placeholder="如「2025-2026 第二学期」" />
          </Form.Item>
          <Form.Item
            name="date_range"
            label="日期范围"
            rules={[{ required: true, message: '请选择日期范围' }]}
          >
            <RangePicker style={{ width: '100%' }} format="YYYY-MM-DD" />
          </Form.Item>
          <Form.Item
            name="first_week_type"
            label="第一周类型"
            rules={[{ required: true, message: '请选择第一周类型' }]}
          >
            <Radio.Group>
              <Radio.Button value="odd">单周</Radio.Button>
              <Radio.Button value="even">双周</Radio.Button>
            </Radio.Group>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

// ════════════════════════════════════════
// Tab 2: 时间段配置（周视图）
// ════════════════════════════════════════
function TimeSlotTab() {
  const [semesters, setSemesters] = useState<SemesterInfo[]>([]);
  const [selectedSemesterId, setSelectedSemesterId] = useState<string>('');
  const [timeSlots, setTimeSlots] = useState<TimeSlotInfo[]>([]);
  const [loading, setLoading] = useState(false);

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

  const fetchTimeSlots = useCallback(async () => {
    if (!selectedSemesterId) return;
    setLoading(true);
    try {
      const { data } = await timeSlotApi.list({ semester_id: selectedSemesterId });
      const raw = data.data;
      setTimeSlots(Array.isArray(raw) ? raw : (raw as unknown as { list: TimeSlotInfo[] }).list ?? []);
    } catch {
      message.error('获取时间段列表失败');
    } finally {
      setLoading(false);
    }
  }, [selectedSemesterId]);

  useEffect(() => { fetchSemesters(); }, [fetchSemesters]);
  useEffect(() => { fetchTimeSlots(); }, [fetchTimeSlots]);

  const currentSemester = semesters.find((s) => s.id === selectedSemesterId);

  return (
    <div>
      <Flex gap={12} align="center" style={{ marginBottom: 16 }}>
        <Text type="secondary">选择学期：</Text>
        <Select
          style={{ width: 260 }}
          value={selectedSemesterId || undefined}
          onChange={setSelectedSemesterId}
          options={semesters.map((s) => ({
            label: s.name + (s.is_active ? ' (当前)' : ''),
            value: s.id,
          }))}
          placeholder="请选择学期"
        />
      </Flex>
      <WeekGridView
        timeSlots={timeSlots}
        loading={loading}
        semester={currentSemester}
        onRefresh={fetchTimeSlots}
      />
    </div>
  );
}

// ════════════════════════════════════════
// Tab 3: 值班地点
// ════════════════════════════════════════
function LocationTab() {
  const [locations, setLocations] = useState<LocationInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<LocationInfo | null>(null);
  const [form] = Form.useForm();
  const [submitLoading, setSubmitLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await locationApi.list({ include_inactive: true });
      const raw = data.data;
      setLocations(Array.isArray(raw) ? raw : (raw as unknown as { list: LocationInfo[] }).list ?? []);
    } catch {
      message.error('获取地点列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (item: LocationInfo) => {
    setEditing(item);
    form.setFieldsValue({
      name: item.name,
      address: item.address,
      is_default: item.is_default,
      is_active: item.is_active,
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitLoading(true);

      if (editing) {
        const updateData: UpdateLocationRequest = {
          name: values.name,
          address: values.address,
          is_default: values.is_default,
          is_active: values.is_active,
        };
        await locationApi.update(editing.id, updateData);
        message.success('地点已更新');
      } else {
        const createData: CreateLocationRequest = {
          name: values.name,
          address: values.address,
          is_default: values.is_default,
        };
        await locationApi.create(createData);
        message.success('地点已创建');
      }
      setModalOpen(false);
      fetchData();
    } catch (err) {
      showError(err, editing ? '更新地点失败' : '创建地点失败');
    } finally {
      setSubmitLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await locationApi.delete(id);
      message.success('地点已删除');
      fetchData();
    } catch (err) {
      showError(err, '删除失败');
    }
  };

  const columns: ColumnsType<LocationInfo> = [
    {
      title: '地点名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record) => (
        <Space>
          <Text strong>{name}</Text>
          {record.is_default && (
            <Tooltip title="默认地点">
              <StarFilled style={{ color: '#faad14' }} />
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: '地址',
      dataIndex: 'address',
      key: 'address',
      ellipsis: true,
      render: (addr: string | undefined) => addr ?? <Text type="secondary">—</Text>,
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
      width: 140,
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>
            编辑
          </Button>
          <ConfirmAction
            title="确认删除此地点？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </ConfirmAction>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Flex justify="flex-end" style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建地点
        </Button>
      </Flex>
      <Table<LocationInfo>
        rowKey="id"
        columns={columns}
        dataSource={locations}
        loading={loading}
        pagination={false}
      />
      <Modal
        title={editing ? '编辑地点' : '新建地点'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmit}
        confirmLoading={submitLoading}
        okText={editing ? '保存' : '创建'}
        cancelText="取消"
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label="地点名称"
            rules={[
              { required: true, message: '请输入地点名称' },
              { min: 2, max: 100, message: '名称长度 2-100 个字符' },
            ]}
          >
            <Input placeholder="如「图书馆 B2」" />
          </Form.Item>
          <Form.Item
            name="address"
            label="地址"
            rules={[{ max: 200, message: '地址最多 200 个字符' }]}
          >
            <Input placeholder="详细地址（可选）" />
          </Form.Item>
          <Form.Item name="is_default" label="设为默认地点" valuePropName="checked" initialValue={false}>
            <Switch
              checkedChildren={<StarFilled />}
              unCheckedChildren={<StarOutlined />}
            />
          </Form.Item>
          {editing && (
            <Form.Item name="is_active" label="启用状态" valuePropName="checked">
              <Switch
                checkedChildren={<CheckCircleOutlined />}
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </div>
  );
}

// ════════════════════════════════════════
// Tab 4: 排班规则
// ════════════════════════════════════════
function ScheduleRuleTab() {
  const [rules, setRules] = useState<ScheduleRuleInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [toggling, setToggling] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await scheduleRuleApi.list();
      const raw = data.data;
      setRules(Array.isArray(raw) ? raw : (raw as unknown as { list: ScheduleRuleInfo[] }).list ?? []);
    } catch {
      message.error('获取排班规则失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleToggle = async (rule: ScheduleRuleInfo, enabled: boolean) => {
    setToggling(rule.id);
    try {
      await scheduleRuleApi.update(rule.id, { is_enabled: enabled });
      message.success(`「${rule.rule_name}」已${enabled ? '启用' : '禁用'}`);
      fetchData();
    } catch (err) {
      showError(err, '操作失败');
    } finally {
      setToggling(null);
    }
  };

  return (
    <div>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        排班规则由系统预置，可根据需要启用或禁用可配置的规则。
      </Text>
      {loading ? (
        <Table loading={loading} columns={[]} dataSource={[]} />
      ) : rules.length === 0 ? (
        <Empty description="暂无排班规则" />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {rules.map((rule) => (
            <div
              key={rule.id}
              style={{
                padding: '16px 20px',
                borderRadius: 8,
                border: '1px solid #f0f0f0',
                background: rule.is_enabled ? '#fff' : '#fafafa',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                transition: 'all 0.2s',
              }}
            >
              <div style={{ flex: 1 }}>
                <Flex align="center" gap={8}>
                  <SafetyCertificateOutlined
                    style={{ color: rule.is_enabled ? '#1677ff' : '#bbb', fontSize: 16 }}
                  />
                  <Text strong style={{ fontSize: 14 }}>
                    {rule.rule_name}
                  </Text>
                  {!rule.is_configurable && (
                    <Tag bordered={false} color="default" style={{ fontSize: 11 }}>
                      系统内置
                    </Tag>
                  )}
                </Flex>
                {rule.description && (
                  <Text type="secondary" style={{ display: 'block', marginTop: 4, marginLeft: 24 }}>
                    {rule.description}
                  </Text>
                )}
              </div>
              <Tooltip
                title={!rule.is_configurable ? '此规则为系统内置，不可修改' : undefined}
              >
                <Switch
                  checked={rule.is_enabled}
                  disabled={!rule.is_configurable}
                  loading={toggling === rule.id}
                  onChange={(checked) => handleToggle(rule, checked)}
                />
              </Tooltip>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ════════════════════════════════════════
// Tab 5: 系统参数
// ════════════════════════════════════════
function SystemParamTab() {
  const [form] = Form.useForm<UpdateSystemConfigRequest>();
  const [loading, setLoading] = useState(false);
  const [saveLoading, setSaveLoading] = useState(false);
  const [config, setConfig] = useState<SystemConfigInfo | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await systemConfigApi.get();
      setConfig(data.data);
      form.setFieldsValue({
        swap_deadline_hours: data.data.swap_deadline_hours,
        duty_reminder_time: data.data.duty_reminder_time,
        default_location: data.data.default_location,
        sign_in_window_minutes: data.data.sign_in_window_minutes,
        sign_out_window_minutes: data.data.sign_out_window_minutes,
      });
    } catch (err) {
      showError(err, '加载系统参数失败');
    } finally {
      setLoading(false);
    }
  }, [form]);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaveLoading(true);
      await systemConfigApi.update(values);
      message.success('系统参数已保存');
      fetchConfig();
    } catch (err) {
      showError(err, '保存失败');
    } finally {
      setSaveLoading(false);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        layout="vertical"
        style={{ maxWidth: 520 }}
      >
        <Form.Item
          name="swap_deadline_hours"
          label="换班申请截止时间（小时）"
          tooltip="距排班开始前多少小时关闭换班申请"
          rules={[{ required: true, message: '请输入' }]}
        >
          <InputNumber min={0} max={168} addonAfter="小时" style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="duty_reminder_time"
          label="值班提醒时间"
          tooltip="每天在此时间发送值班提醒"
        >
          <Input placeholder="如 08:00" style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="default_location"
          label="默认值班地点"
        >
          <Input placeholder="默认值班地点名称" />
        </Form.Item>

        <Form.Item
          name="sign_in_window_minutes"
          label="签到窗口时间（分钟）"
          tooltip="排班开始前后多少分钟内允许签到"
          rules={[{ required: true, message: '请输入' }]}
        >
          <InputNumber min={0} max={120} addonAfter="分钟" style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="sign_out_window_minutes"
          label="签退窗口时间（分钟）"
          tooltip="排班结束前后多少分钟内允许签退"
          rules={[{ required: true, message: '请输入' }]}
        >
          <InputNumber min={0} max={120} addonAfter="分钟" style={{ width: '100%' }} />
        </Form.Item>

        {config?.updated_at && (
          <Text type="secondary" style={{ display: 'block', marginBottom: 16, fontSize: 12 }}>
            上次更新：{dayjs(config.updated_at).format('YYYY-MM-DD HH:mm:ss')}
          </Text>
        )}

        <Form.Item>
          <Button type="primary" loading={saveLoading} onClick={handleSave}>
            保存配置
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
}

// ════════════════════════════════════════
// 主页面
// ════════════════════════════════════════
export default function ConfigPage() {
  const tabItems = [
    {
      key: 'semester',
      label: (
        <span><CalendarOutlined style={{ marginRight: 6 }} />学期管理</span>
      ),
      children: <SemesterTab />,
    },
    {
      key: 'timeslot',
      label: (
        <span><ClockCircleOutlined style={{ marginRight: 6 }} />时间段配置</span>
      ),
      children: <TimeSlotTab />,
    },
    {
      key: 'location',
      label: (
        <span><EnvironmentOutlined style={{ marginRight: 6 }} />值班地点</span>
      ),
      children: <LocationTab />,
    },
    {
      key: 'rule',
      label: (
        <span><SafetyCertificateOutlined style={{ marginRight: 6 }} />排班规则</span>
      ),
      children: <ScheduleRuleTab />,
    },
    {
      key: 'system',
      label: (
        <span><SettingOutlined style={{ marginRight: 6 }} />系统参数</span>
      ),
      children: <SystemParamTab />,
    },
  ];

  return (
    <div>
      <PageHeader
        title="系统配置"
        description="管理学期、时间段、地点与排班规则等系统级配置"
      />
      <Tabs items={tabItems} defaultActiveKey="semester" />
    </div>
  );
}
