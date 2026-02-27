import { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Typography,
  Spin,
  Statistic,
  Row,
  Col,
  Progress,
  Tag,
  Space,
  message,
} from 'antd';
import {
  UserOutlined,
  TeamOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { userApi } from '@/services/userApi';
import { departmentApi } from '@/services/departmentApi';
import { semesterApi } from '@/services/configApi';
import { timetableApi, scheduleApi } from '@/services';
import type {
  SemesterInfo,
  DepartmentDetail,
  TimetableProgressResponse,
  ScheduleItem,
} from '@/types';

const { Text } = Typography;

export default function AdminDashboard() {
  const [stats, setStats] = useState<{
    userCount: number;
    departmentCount: number;
    currentSemester: SemesterInfo | null;
    activeDepartments: number;
  }>({
    userCount: 0,
    departmentCount: 0,
    currentSemester: null,
    activeDepartments: 0,
  });
  const [progress, setProgress] = useState<TimetableProgressResponse | null>(null);
  const [todayItems, setTodayItems] = useState<ScheduleItem[]>([]);
  const [loading, setLoading] = useState(false);

  const todayDow = dayjs().day() || 7; // 1=Mon ... 7=Sun

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [usersRes, deptsRes, semRes, progRes, scheRes] = await Promise.allSettled([
        userApi.listUsers({ page: 1, page_size: 1 }),
        departmentApi.list({ include_inactive: true }),
        semesterApi.getCurrent(),
        timetableApi.getProgress(),
        scheduleApi.getSchedule(),
      ]);

      const userCount =
        usersRes.status === 'fulfilled'
          ? usersRes.value.data.data.pagination.total
          : 0;

      let departmentCount = 0;
      let activeDepartments = 0;
      if (deptsRes.status === 'fulfilled') {
        const list = deptsRes.value.data.data as DepartmentDetail[];
        departmentCount = list.length;
        activeDepartments = list.filter((d) => d.is_active).length;
      }

      const currentSemester =
        semRes.status === 'fulfilled' ? semRes.value.data.data : null;

      setStats({ userCount, departmentCount, currentSemester, activeDepartments });

      if (progRes.status === 'fulfilled') {
        setProgress(progRes.value.data.data);
      }

      if (scheRes.status === 'fulfilled') {
        const items = scheRes.value.data.data?.items || [];
        setTodayItems(
          items.filter((item) => item.time_slot?.day_of_week === todayDow),
        );
      }
    } catch {
      message.warning('部分数据加载失败');
    } finally {
      setLoading(false);
    }
  }, [todayDow]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return (
    <Spin spinning={loading}>
      {/* 统计卡片 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card variant="borderless" style={{ background: '#f0f5ff' }}>
            <Statistic
              title="用户总数"
              value={stats.userCount}
              prefix={<UserOutlined style={{ color: '#1677ff' }} />}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card variant="borderless" style={{ background: '#f6ffed' }}>
            <Statistic
              title="部门数（启用）"
              value={stats.activeDepartments}
              suffix={`/ ${stats.departmentCount}`}
              prefix={<TeamOutlined style={{ color: '#52c41a' }} />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card variant="borderless" style={{ background: '#fff7e6' }}>
            <Statistic
              title="当前学期"
              value={stats.currentSemester?.name ?? '未设置'}
              prefix={<CalendarOutlined style={{ color: '#fa8c16' }} />}
              valueStyle={{ color: '#fa8c16', fontSize: stats.currentSemester ? 18 : 16 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card variant="borderless" style={{ background: '#f9f0ff' }}>
            <Statistic
              title="学期状态"
              value={
                stats.currentSemester
                  ? `${stats.currentSemester.start_date} ~ ${stats.currentSemester.end_date}`
                  : '—'
              }
              prefix={<ClockCircleOutlined style={{ color: '#722ed1' }} />}
              valueStyle={{ color: '#722ed1', fontSize: 14 }}
            />
          </Card>
        </Col>
      </Row>

      {/* 时间表提交进度 */}
      {progress && (
        <Card title="时间表提交进度" style={{ marginTop: 16 }} size="small">
          <Row gutter={16} align="middle">
            <Col span={6}>
              <Progress
                type="circle"
                percent={Math.round(progress.progress * 100)}
                size={80}
                format={() => `${progress.submitted}/${progress.total}`}
              />
            </Col>
            <Col span={18}>
              <Row gutter={[8, 8]}>
                {progress.departments.map((dep) => (
                  <Col span={12} key={dep.department_id}>
                    <Text style={{ fontSize: 12 }}>{dep.department_name}</Text>
                    <Progress
                      percent={Math.round(dep.progress * 100)}
                      size="small"
                      format={() => `${dep.submitted}/${dep.total}`}
                    />
                  </Col>
                ))}
              </Row>
            </Col>
          </Row>
        </Card>
      )}

      {/* 今日排班 */}
      <Card title="今日排班" style={{ marginTop: 16 }} size="small">
        {todayItems.length > 0 ? (
          <Space wrap>
            {todayItems.map((item) => (
              <Tag key={item.id} color="blue">
                {item.time_slot
                  ? `${item.time_slot.start_time.slice(0, 5)}-${item.time_slot.end_time.slice(0, 5)}`
                  : ''}
                {' '}
                {item.member?.name ?? '待定'}
                {item.location ? ` @ ${item.location.name}` : ''}
              </Tag>
            ))}
          </Space>
        ) : (
          <Text type="secondary">今日无排班</Text>
        )}
      </Card>
    </Spin>
  );
}
