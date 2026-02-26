import { Card, Typography, Progress, Alert, Flex, Button } from 'antd';
import { useNavigate } from 'react-router-dom';
import type { AutoScheduleResponse } from '@/types';

const { Text, Title } = Typography;

interface ScheduleResultPanelProps {
  result: AutoScheduleResponse | null;
}

/**
 * 自动排班结果展示面板
 */
export default function ScheduleResultPanel({
  result,
}: ScheduleResultPanelProps) {
  const navigate = useNavigate();

  if (!result) return null;

  const { total_slots, filled_slots, warnings, schedule } = result;
  const fillRate = total_slots > 0 ? Math.round((filled_slots / total_slots) * 100) : 0;

  return (
    <Card title="排班结果" size="small">
      <Flex gap={24} align="center" style={{ marginBottom: 16 }}>
        <Progress
          type="circle"
          percent={fillRate}
          size={80}
          format={() => `${filled_slots}/${total_slots}`}
        />
        <div>
          <Title level={4} style={{ margin: 0 }}>
            已填充 {filled_slots}/{total_slots} 个时段
          </Title>
          <Text type="secondary">
            排班表状态：{schedule.status === 'published' ? '已发布' : '草稿'}
          </Text>
        </div>
      </Flex>

      {/* 警告 */}
      {warnings && warnings.length > 0 && (
        <div style={{ marginBottom: 16 }}>
          {warnings.map((w, idx) => (
            <Alert
              key={idx}
              message={w}
              type="warning"
              showIcon
              style={{ marginBottom: 8 }}
            />
          ))}
        </div>
      )}

      <Flex gap={8}>
        <Button type="primary" onClick={() => navigate('/schedule')}>
          查看排班结果
        </Button>
        <Button onClick={() => navigate('/schedule/adjust')}>
          前往手动调整
        </Button>
      </Flex>
    </Card>
  );
}
