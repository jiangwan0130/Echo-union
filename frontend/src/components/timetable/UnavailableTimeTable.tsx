import { Table, Button, Space, Tag, Popconfirm } from 'antd';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';
import type { UnavailableTime } from '@/types';

const WEEKDAY_LABELS: Record<number, string> = {
  1: '周一', 2: '周二', 3: '周三', 4: '周四', 5: '周五', 6: '周六', 7: '周日',
};

const REPEAT_LABELS: Record<string, string> = {
  weekly: '每周',
  biweekly: '隔周',
  once: '仅一次',
};

const WEEK_TYPE_LABELS: Record<string, string> = {
  all: '所有周',
  odd: '单周',
  even: '双周',
};

interface UnavailableTimeTableProps {
  items: UnavailableTime[];
  loading?: boolean;
  onEdit: (item: UnavailableTime) => void;
  onDelete: (id: string) => void;
}

/** 格式化时间 "08:10:00" → "08:10" */
const fmt = (t: string) => t.replace(/:\d{2}$/, '');

/**
 * 不可用时间列表表格
 */
export default function UnavailableTimeTable({
  items,
  loading = false,
  onEdit,
  onDelete,
}: UnavailableTimeTableProps) {
  const columns = [
    {
      title: '星期',
      dataIndex: 'day_of_week',
      key: 'day_of_week',
      width: 80,
      render: (d: number) => WEEKDAY_LABELS[d] || `星期${d}`,
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
      width: 100,
      render: (t: string) => fmt(t),
    },
    {
      title: '结束时间',
      dataIndex: 'end_time',
      key: 'end_time',
      width: 100,
      render: (t: string) => fmt(t),
    },
    {
      title: '重复类型',
      dataIndex: 'repeat_type',
      key: 'repeat_type',
      width: 100,
      render: (v: string) => <Tag>{REPEAT_LABELS[v] || v}</Tag>,
    },
    {
      title: '周类型',
      dataIndex: 'week_type',
      key: 'week_type',
      width: 100,
      render: (v: string) => WEEK_TYPE_LABELS[v] || v,
    },
    {
      title: '备注',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
      render: (v: string) => v || '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_: unknown, record: UnavailableTime) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => onEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除此不可用时间？"
            onConfirm={() => onDelete(record.id)}
          >
            <Button type="text" danger size="small" icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={items}
      rowKey="id"
      loading={loading}
      size="small"
      pagination={false}
      locale={{ emptyText: '暂无不可用时间' }}
    />
  );
}
