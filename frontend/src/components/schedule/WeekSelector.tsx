import { Button, Space, Typography } from 'antd';
import { LeftOutlined, RightOutlined } from '@ant-design/icons';

const { Text } = Typography;

interface WeekSelectorProps {
  weekNumber: number;
  maxWeek?: number;
  onChange: (week: number) => void;
}

/**
 * 周选择器 ◀ 第N周 ▶
 */
export default function WeekSelector({
  weekNumber,
  maxWeek = 20,
  onChange,
}: WeekSelectorProps) {
  return (
    <Space size={4}>
      <Button
        type="text"
        size="small"
        icon={<LeftOutlined />}
        disabled={weekNumber <= 1}
        onClick={() => onChange(weekNumber - 1)}
      />
      <Text strong style={{ minWidth: 60, textAlign: 'center', display: 'inline-block' }}>
        第{weekNumber}周
      </Text>
      <Button
        type="text"
        size="small"
        icon={<RightOutlined />}
        disabled={weekNumber >= maxWeek}
        onClick={() => onChange(weekNumber + 1)}
      />
    </Space>
  );
}
