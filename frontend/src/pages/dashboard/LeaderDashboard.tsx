import { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Spin,
  Statistic,
  Row,
  Col,
  Table,
  Tag,
  Empty,
} from 'antd';
import {
  TeamOutlined,
  CalendarOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useAuthStore } from '@/stores';
import { timetableApi, showError } from '@/services';
import type { DepartmentProgressResponse } from '@/types';

export default function LeaderDashboard() {
  const { user } = useAuthStore();
  const departmentId = user?.department?.id;

  const [progress, setProgress] = useState<DepartmentProgressResponse | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    if (!departmentId) return;
    setLoading(true);
    try {
      const { data } = await timetableApi.getDepartmentProgress(departmentId);
      setProgress(data.data);
    } catch (err) {
      showError(err, '加载部门进度失败');
    } finally {
      setLoading(false);
    }
  }, [departmentId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  if (!departmentId) {
    return <Empty description="未分配部门" />;
  }

  const columns = [
    { title: '姓名', dataIndex: 'name', key: 'name' },
    { title: '学号', dataIndex: 'student_id', key: 'student_id' },
    {
      title: '提交状态',
      dataIndex: 'timetable_status',
      key: 'status',
      render: (v: string) => (
        <Tag color={v === 'submitted' ? 'green' : 'orange'}>
          {v === 'submitted' ? '已提交' : '未提交'}
        </Tag>
      ),
    },
    {
      title: '提交时间',
      dataIndex: 'submitted_at',
      key: 'submitted_at',
      render: (v?: string) => (v ? dayjs(v).format('MM-DD HH:mm') : '—'),
    },
  ];

  return (
    <Spin spinning={loading}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={8}>
          <Card variant="borderless" style={{ background: '#f0f5ff' }}>
            <Statistic
              title="部门成员"
              value={progress?.total ?? 0}
              prefix={<TeamOutlined style={{ color: '#1677ff' }} />}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card variant="borderless" style={{ background: '#f6ffed' }}>
            <Statistic
              title="已提交时间表"
              value={progress?.submitted ?? 0}
              suffix={`/ ${progress?.total ?? 0}`}
              prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={8}>
          <Card variant="borderless" style={{ background: '#fff7e6' }}>
            <Statistic
              title="提交率"
              value={Math.round((progress?.progress ?? 0) * 100)}
              suffix="%"
              prefix={<CalendarOutlined style={{ color: '#fa8c16' }} />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
      </Row>

      {progress?.members && (
        <Card title="成员提交明细" style={{ marginTop: 16 }} size="small">
          <Table
            dataSource={progress.members}
            columns={columns}
            rowKey="user_id"
            size="small"
            pagination={false}
          />
        </Card>
      )}
    </Spin>
  );
}
