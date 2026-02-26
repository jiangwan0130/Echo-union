import { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Typography,
  Flex,
  Spin,
  Statistic,
  Row,
  Col,
  Progress,
  Table,
  Tag,
  Empty,
  Space,
} from 'antd';
import {
  UserOutlined,
  TeamOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  ScheduleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { useAuthStore } from '@/stores';
import { userApi } from '@/services/userApi';
import { departmentApi } from '@/services/departmentApi';
import { semesterApi } from '@/services/configApi';
import { timetableApi, scheduleApi, showError } from '@/services';
import type {
  SemesterInfo,
  DepartmentDetail,
  TimetableProgressResponse,
  DepartmentProgressResponse,
  ScheduleItem,
  MyTimetableResponse,
} from '@/types';

const { Title, Text } = Typography;

// ── 管理员仪表板 ──
function AdminDashboard() {
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
        const raw = deptsRes.value.data.data;
        const list: DepartmentDetail[] = Array.isArray(raw)
          ? raw
          : (raw as unknown as { list: DepartmentDetail[] }).list ?? [];
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
      /* silently ignore */
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

// ── 负责人仪表板 ──
function LeaderDashboard() {
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

// ── 成员仪表板 ──
function MemberDashboard() {
  const [timetable, setTimetable] = useState<MyTimetableResponse | null>(null);
  const [nextShifts, setNextShifts] = useState<ScheduleItem[]>([]);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [ttRes, scheRes] = await Promise.allSettled([
        timetableApi.getMyTimetable(),
        scheduleApi.getMySchedule(),
      ]);

      if (ttRes.status === 'fulfilled') {
        setTimetable(ttRes.value.data.data);
      }

      if (scheRes.status === 'fulfilled') {
        const items = scheRes.value.data.data?.items || [];
        // 按周+时段排序，取前5条排班
        const sorted = items
          .filter((i) => i.time_slot)
          .sort((a, b) => {
            const weekDiff = a.week_number - b.week_number;
            if (weekDiff !== 0) return weekDiff;
            return (a.time_slot!.day_of_week - b.time_slot!.day_of_week) ||
              a.time_slot!.start_time.localeCompare(b.time_slot!.start_time);
          });
        setNextShifts(sorted.slice(0, 5));
      }
    } catch {
      /* silently ignore */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const ttStatus = timetable?.submit_status;

  return (
    <Spin spinning={loading}>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12}>
          <Card
            variant="borderless"
            style={{ background: ttStatus === 'submitted' ? '#f6ffed' : '#fff7e6', cursor: 'pointer' }}
            onClick={() => navigate('/timetable')}
          >
            <Statistic
              title="时间表状态"
              value={ttStatus === 'submitted' ? '已提交' : '未提交'}
              prefix={
                ttStatus === 'submitted' ? (
                  <CheckCircleOutlined style={{ color: '#52c41a' }} />
                ) : (
                  <ClockCircleOutlined style={{ color: '#fa8c16' }} />
                )
              }
              valueStyle={{ color: ttStatus === 'submitted' ? '#52c41a' : '#fa8c16' }}
            />
            {timetable?.submitted_at && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                提交于 {dayjs(timetable.submitted_at).format('YYYY-MM-DD HH:mm')}
              </Text>
            )}
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card variant="borderless" style={{ background: '#f0f5ff' }}>
            <Statistic
              title="课程数 / 不可用时段"
              value={timetable?.courses?.length ?? 0}
              suffix={`/ ${timetable?.unavailable?.length ?? 0}`}
              prefix={<ScheduleOutlined style={{ color: '#1677ff' }} />}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
      </Row>

      <Card title="我的近期排班" style={{ marginTop: 16 }} size="small">
        {nextShifts.length > 0 ? (
          <Space direction="vertical" style={{ width: '100%' }}>
            {nextShifts.map((item) => (
              <Flex key={item.id} justify="space-between" align="center">
                <Text>
                  <Tag color="blue">第{item.week_number}周</Tag>
                  {item.time_slot
                    ? `${{ 1: '周一', 2: '周二', 3: '周三', 4: '周四', 5: '周五' }[item.time_slot.day_of_week] ?? ''} ${item.time_slot.start_time.slice(0, 5)}-${item.time_slot.end_time.slice(0, 5)}`
                    : ''}
                </Text>
                <Text type="secondary">
                  {item.location?.name ?? ''}
                </Text>
              </Flex>
            ))}
          </Space>
        ) : (
          <Text type="secondary">暂无排班</Text>
        )}
      </Card>
    </Spin>
  );
}

// ── 主页面 ──
export default function DashboardPage() {
  const { user } = useAuthStore();
  const role = user?.role;

  return (
    <div>
      <Flex justify="space-between" align="flex-start" style={{ marginBottom: 24 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            欢迎回来，{user?.name}
          </Title>
          <Text type="secondary">
            角色：{role === 'admin' ? '管理员' : role === 'leader' ? '负责人' : '成员'}
            {user?.department?.name ? ` | 部门：${user.department.name}` : ''}
          </Text>
        </div>
      </Flex>

      {role === 'admin' && <AdminDashboard />}
      {role === 'leader' && <LeaderDashboard />}
      {role === 'member' && <MemberDashboard />}
    </div>
  );
}
