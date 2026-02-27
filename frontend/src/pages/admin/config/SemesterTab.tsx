import { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  DatePicker,
  Radio,
  Tag,
  message,
  Typography,
  Flex,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { ConfirmAction } from '@/components/common';
import { semesterApi } from '@/services/configApi';
import { showError } from '@/services/errorHandler';
import type {
  SemesterInfo,
  CreateSemesterRequest,
  UpdateSemesterRequest,
} from '@/types';

const { Text } = Typography;
const { RangePicker } = DatePicker;

export default function SemesterTab() {
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
      setSemesters(data.data as SemesterInfo[]);
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
