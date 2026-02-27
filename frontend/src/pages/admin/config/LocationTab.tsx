import { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Switch,
  Tooltip,
  message,
  Typography,
  Flex,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  StarFilled,
  StarOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { StatusTag, ConfirmAction } from '@/components/common';
import { locationApi } from '@/services/configApi';
import { showError } from '@/services/errorHandler';
import type {
  LocationInfo,
  CreateLocationRequest,
  UpdateLocationRequest,
} from '@/types';

const { Text } = Typography;

export default function LocationTab() {
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
      setLocations(data.data as LocationInfo[]);
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
