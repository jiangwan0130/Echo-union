import { useState, useEffect, useCallback } from 'react';
import { Drawer, Table, Tag, Typography } from 'antd';
import { scheduleApi, showError } from '@/services';
import type { ScheduleChangeLog, ChangeLogListParams } from '@/types';

const { Text } = Typography;

interface ChangeLogDrawerProps {
  open: boolean;
  onClose: () => void;
  scheduleId: string;
}

/**
 * 排班变更记录抽屉
 */
export default function ChangeLogDrawer({
  open,
  onClose,
  scheduleId,
}: ChangeLogDrawerProps) {
  const [loading, setLoading] = useState(false);
  const [logs, setLogs] = useState<ScheduleChangeLog[]>([]);
  const [pagination, setPagination] = useState({ page: 1, page_size: 20, total: 0 });

  const fetchLogs = useCallback(async () => {
    if (!scheduleId) return;
    setLoading(true);
    try {
      const params: ChangeLogListParams = {
        schedule_id: scheduleId,
        page: pagination.page,
        page_size: pagination.page_size,
      };
      const { data } = await scheduleApi.listChangeLogs(params);
      setLogs(data.data?.list || []);
      setPagination((prev) => ({
        ...prev,
        total: data.data?.pagination?.total || 0,
      }));
    } catch (err) {
      showError(err, '加载变更记录失败');
    } finally {
      setLoading(false);
    }
  }, [scheduleId, pagination.page, pagination.page_size]);

  useEffect(() => {
    if (open && scheduleId) {
      fetchLogs();
    }
  }, [open, scheduleId, fetchLogs]);

  const columns = [
    {
      title: '变更类型',
      dataIndex: 'change_type',
      key: 'change_type',
      width: 100,
      render: (v: string) => (
        <Tag color={v === 'manual_adjust' ? 'blue' : 'orange'}>{v}</Tag>
      ),
    },
    {
      title: '原值班人',
      dataIndex: 'original_member_name',
      key: 'original',
      width: 100,
      render: (v: string) => v || '-',
    },
    {
      title: '新值班人',
      dataIndex: 'new_member_name',
      key: 'new',
      width: 100,
      render: (v: string) => v || '-',
    },
    {
      title: '原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
      render: (v: string) => v || '-',
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString(),
    },
  ];

  return (
    <Drawer
      title="排班变更记录"
      open={open}
      onClose={onClose}
      width={640}
    >
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        以下为排班表发布后的所有修改记录。
      </Text>
      <Table
        columns={columns}
        dataSource={logs}
        rowKey="id"
        loading={loading}
        size="small"
        pagination={{
          total: pagination.total,
          current: pagination.page,
          pageSize: pagination.page_size,
          showSizeChanger: false,
          onChange: (page) => setPagination((p) => ({ ...p, page })),
        }}
      />
    </Drawer>
  );
}
