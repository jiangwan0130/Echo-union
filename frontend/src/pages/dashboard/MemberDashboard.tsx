import { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Typography,
  Flex,
  Spin,
  Statistic,
  Row,
  Col,
  Tag,
  Space,
  message,
} from 'antd';
import {
  ClockCircleOutlined,
  CheckCircleOutlined,
  ScheduleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { timetableApi, scheduleApi } from '@/services';
import type { MyTimetableResponse, ScheduleItem } from '@/types';

const { Text } = Typography;

export default function MemberDashboard() {
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
      message.warning('部分数据加载失败');
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
                    ? `${{ 1: '周一', 2: '周二', 3: '周三', 4: '周四', 5: '周五', 6: '周六', 7: '周日' }[item.time_slot.day_of_week] ?? ''} ${item.time_slot.start_time.slice(0, 5)}-${item.time_slot.end_time.slice(0, 5)}`
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
