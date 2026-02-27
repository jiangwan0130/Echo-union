import { lazy, Suspense, useEffect } from 'react';
import { Card, Space, Spin, Result, Alert, Typography } from 'antd';
import {
  TeamOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ScheduleOutlined,
  EditOutlined,
} from '@ant-design/icons';
import { useAppStore, useAuthStore } from '@/stores';
import type { PendingTodoItem } from '@/types';

const TimetablePage = lazy(() => import('@/pages/timetable/TimetablePage'));
const SchedulePage = lazy(() => import('@/pages/schedule/SchedulePage'));
const ProgressPage = lazy(() => import('@/pages/admin/progress/ProgressPage'));

const { Text } = Typography;

export default function LeaderWorkbench() {
  const {
    currentSemester,
    currentPhase,
    pendingTodos,
    fetchPendingTodos,
    fetchCurrentSemester,
  } = useAppStore();
  const { user } = useAuthStore();

  useEffect(() => {
    fetchCurrentSemester();
    fetchPendingTodos();
  }, [fetchCurrentSemester, fetchPendingTodos]);

  if (!currentSemester) {
    return (
      <Result
        status="info"
        title="暂无排班任务"
        subTitle="当前没有活跃的学期安排"
      />
    );
  }

  const iconMap: Record<string, React.ReactNode> = {
    submit_timetable: <EditOutlined style={{ color: '#faad14' }} />,
    timetable_submitted: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
    waiting_schedule: <ClockCircleOutlined style={{ color: '#1677ff' }} />,
    schedule_published: <ScheduleOutlined style={{ color: '#52c41a' }} />,
  };

  const colorMap: Record<string, 'warning' | 'success' | 'info'> = {
    submit_timetable: 'warning',
    timetable_submitted: 'success',
    waiting_schedule: 'info',
    schedule_published: 'success',
  };

  return (
    <div>
      {/* 待办卡片 */}
      <Space direction="vertical" style={{ width: '100%', marginBottom: 16 }}>
        {pendingTodos.map((todo, i) => (
          <Alert
            key={i}
            type={colorMap[todo.type] ?? 'info'}
            message={
              <Space>
                {iconMap[todo.type]}
                <Text strong>{todo.title}</Text>
              </Space>
            }
            description={todo.message}
            showIcon={false}
            style={{ borderRadius: 8 }}
          />
        ))}
      </Space>

      {/* 本部门提交进度（collecting 阶段） */}
      {currentPhase === 'collecting' && (
        <Card
          title={
            <Space>
              <TeamOutlined />
              本部门提交进度
            </Space>
          }
          style={{ marginBottom: 16 }}
        >
          <Suspense fallback={<Spin />}>
            <ProgressPage />
          </Suspense>
        </Card>
      )}

      {/* 时间表填写（如果 leader 本人也需要提交） */}
      {currentPhase === 'collecting' &&
        pendingTodos.some((t) => t.type === 'submit_timetable') && (
          <Card
            title={
              <Space>
                <EditOutlined />
                我的时间表
              </Space>
            }
            style={{ marginBottom: 16 }}
          >
            <Suspense fallback={<Spin />}>
              <TimetablePage />
            </Suspense>
          </Card>
        )}

      {/* 排班结果 */}
      {currentPhase === 'published' && (
        <Card
          title={
            <Space>
              <ScheduleOutlined />
              排班结果
            </Space>
          }
        >
          <Suspense fallback={<Spin />}>
            <SchedulePage />
          </Suspense>
        </Card>
      )}
    </div>
  );
}
