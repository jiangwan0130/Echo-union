import { useEffect, lazy, Suspense } from 'react';
import { Card, Result, Spin, Button, Alert, Typography, Space, Tag } from 'antd';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  ScheduleOutlined,
  EditOutlined,
} from '@ant-design/icons';
import { useAppStore } from '@/stores';
import type { PendingTodoItem } from '@/types';

const TimetablePage = lazy(() => import('@/pages/timetable/TimetablePage'));
const SchedulePage = lazy(() => import('@/pages/schedule/SchedulePage'));

const { Text } = Typography;

export default function MemberWorkbench() {
  const {
    currentSemester,
    currentPhase,
    pendingTodos,
    fetchPendingTodos,
    fetchCurrentSemester,
  } = useAppStore();

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

  return (
    <div>
      {/* 待办卡片区 */}
      <TodoCards todos={pendingTodos} phase={currentPhase} />

      {/* 时间表填写区（collecting 阶段且需要提交时内嵌显示） */}
      {currentPhase === 'collecting' &&
        pendingTodos.some((t) => t.type === 'submit_timetable') && (
          <Card
            title={
              <Space>
                <EditOutlined />
                我的时间表
              </Space>
            }
            style={{ marginTop: 16 }}
          >
            <Suspense fallback={<Spin />}>
              <TimetablePage />
            </Suspense>
          </Card>
        )}

      {/* 排班结果区（published 阶段显示） */}
      {currentPhase === 'published' && (
        <Card
          title={
            <Space>
              <ScheduleOutlined />
              我的排班
            </Space>
          }
          style={{ marginTop: 16 }}
        >
          <Suspense fallback={<Spin />}>
            <SchedulePage />
          </Suspense>
        </Card>
      )}
    </div>
  );
}

function TodoCards({
  todos,
  phase,
}: {
  todos: PendingTodoItem[];
  phase: string | null;
}) {
  if (!todos.length && !phase) {
    return null;
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
    <Space direction="vertical" style={{ width: '100%' }}>
      {todos.map((todo, i) => (
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
  );
}
