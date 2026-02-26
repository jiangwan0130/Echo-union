import { Tag, Typography, Tooltip } from 'antd';
import { UserOutlined, EnvironmentOutlined } from '@ant-design/icons';
import type { ScheduleItem } from '@/types';

const { Text } = Typography;

interface ScheduleItemCardProps {
  items: ScheduleItem[];
  editable?: boolean;
  onClick?: () => void;
}

/**
 * 单时段排班卡片 — 展示该时段的所有值班人员和地点
 */
export default function ScheduleItemCard({
  items,
  editable = false,
  onClick,
}: ScheduleItemCardProps) {
  if (items.length === 0) return null;

  const location = items[0]?.location;

  return (
    <div
      style={{
        padding: '6px 8px',
        borderRadius: 6,
        background: '#e6f4ff',
        border: '1px solid #91caff',
        cursor: editable ? 'pointer' : 'default',
        transition: 'all 0.2s',
        height: '100%',
      }}
      onClick={editable ? onClick : undefined}
      onMouseEnter={(e) => {
        if (editable) {
          e.currentTarget.style.boxShadow = '0 2px 8px rgba(0,0,0,0.1)';
        }
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.boxShadow = 'none';
      }}
    >
      {items.map((item) => (
        <Tooltip
          key={item.id}
          title={`${item.member?.name} · ${item.member?.department?.name || ''}`}
        >
          <Tag
            icon={<UserOutlined />}
            color="blue"
            style={{ marginBottom: 2, fontSize: 11 }}
          >
            {item.member?.name || '未分配'}
          </Tag>
        </Tooltip>
      ))}
      {location && (
        <div style={{ marginTop: 2 }}>
          <Text style={{ fontSize: 10, color: '#8c8c8c' }}>
            <EnvironmentOutlined /> {location.name}
          </Text>
        </div>
      )}
    </div>
  );
}
