import { useMemo } from 'react';
import { Typography, Empty } from 'antd';
import { WarningOutlined } from '@ant-design/icons';
import ScheduleItemCard from './ScheduleItemCard';
import type { ScheduleItem, TimeSlotBrief } from '@/types';

const { Text } = Typography;

const WEEKDAYS = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
];

/** 规范化时间 "08:10:00" → "08:10" */
const fmt = (t: string) => t.replace(/:\d{2}$/, '');

interface ScheduleWeekGridProps {
  items: ScheduleItem[];
  weekNumber: number;
  editable?: boolean;
  /** 点击某个时段卡片时回调（传入该时段的 items 和 timeSlot） */
  onSlotClick?: (timeSlot: TimeSlotBrief, slotItems: ScheduleItem[]) => void;
}

/**
 * 排班周视图网格
 * 按时间段和星期展示排班信息，可切换只读/可编辑模式
 */
export default function ScheduleWeekGrid({
  items,
  weekNumber,
  editable = false,
  onSlotClick,
}: ScheduleWeekGridProps) {
  // 按当前周过滤
  const weekItems = useMemo(
    () => items.filter((item) => item.week_number === weekNumber),
    [items, weekNumber],
  );

  // 提取所有唯一时间段（按 start_time 排序）
  const { timeRows, slotMap } = useMemo(() => {
    const timeSlotMap = new Map<string, TimeSlotBrief>();
    weekItems.forEach((item) => {
      if (item.time_slot) {
        const key = `${fmt(item.time_slot.start_time)}-${fmt(item.time_slot.end_time)}`;
        if (!timeSlotMap.has(key)) {
          timeSlotMap.set(key, item.time_slot);
        }
      }
    });

    const rows = [...timeSlotMap.entries()].sort(([a], [b]) => a.localeCompare(b));

    // 构建映射: (timeKey, dayOfWeek) → ScheduleItem[]
    const map = new Map<string, ScheduleItem[]>();
    weekItems.forEach((item) => {
      if (!item.time_slot) return;
      const timeKey = `${fmt(item.time_slot.start_time)}-${fmt(item.time_slot.end_time)}`;
      const cellKey = `${timeKey}_${item.time_slot.day_of_week}`;
      const arr = map.get(cellKey) || [];
      arr.push(item);
      map.set(cellKey, arr);
    });

    return { timeRows: rows, slotMap: map };
  }, [weekItems]);

  if (timeRows.length === 0) {
    return (
      <Empty
        description={`第${weekNumber}周暂无排班数据`}
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      />
    );
  }

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: '90px repeat(5, 1fr)',
        gap: 1,
        background: '#f0f0f0',
        borderRadius: 8,
        overflow: 'hidden',
        border: '1px solid #f0f0f0',
      }}
    >
      {/* 表头 */}
      <div style={headerStyle}>时间</div>
      {WEEKDAYS.map((day) => (
        <div key={day.value} style={headerStyle}>
          {day.label}
        </div>
      ))}

      {/* 数据行 */}
      {timeRows.map(([timeKey, timeSlot]) => [
        <div key={`t-${timeKey}`} style={timeCellStyle}>
          <Text style={{ fontSize: 12, fontWeight: 500 }}>
            {fmt(timeSlot.start_time)}
          </Text>
          <Text type="secondary" style={{ fontSize: 10 }}>~</Text>
          <Text style={{ fontSize: 12, fontWeight: 500 }}>
            {fmt(timeSlot.end_time)}
          </Text>
          <Text type="secondary" style={{ fontSize: 10, marginTop: 2 }}>
            {timeSlot.name}
          </Text>
        </div>,

        ...WEEKDAYS.map((day) => {
          const cellKey = `${timeKey}_${day.value}`;
          const cellItems = slotMap.get(cellKey) || [];
          const isEmpty = cellItems.length === 0;
          // 检查该天是否应该有排班（通过检查所有items中是否有该day_of_week的时段）
          const hasTimeSlotForDay = weekItems.some(
            (it) => it.time_slot?.day_of_week === day.value,
          );

          return (
            <div
              key={cellKey}
              style={{
                background: '#fff',
                padding: 4,
                minHeight: 70,
                display: 'flex',
                alignItems: 'stretch',
              }}
            >
              {!isEmpty ? (
                <div style={{ flex: 1 }}>
                  <ScheduleItemCard
                    items={cellItems}
                    editable={editable}
                    onClick={() => {
                      if (editable && onSlotClick) {
                        // 找到实际的 timeSlot（该天+该时间段的）
                        const realSlot =
                          cellItems[0]?.time_slot || timeSlot;
                        onSlotClick(realSlot, cellItems);
                      }
                    }}
                  />
                </div>
              ) : hasTimeSlotForDay && editable ? (
                <div
                  style={{
                    flex: 1,
                    border: '2px dashed #ff4d4f',
                    borderRadius: 6,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: 'pointer',
                    background: '#fff1f0',
                  }}
                  onClick={() => {
                    if (onSlotClick) {
                      onSlotClick(timeSlot, []);
                    }
                  }}
                >
                  <Text type="danger" style={{ fontSize: 11 }}>
                    <WarningOutlined /> 空缺
                  </Text>
                </div>
              ) : (
                <div style={{ flex: 1 }} />
              )}
            </div>
          );
        }),
      ])}
    </div>
  );
}

const headerStyle: React.CSSProperties = {
  background: '#fafafa',
  padding: '8px 6px',
  textAlign: 'center',
  fontWeight: 600,
  fontSize: 13,
};

const timeCellStyle: React.CSSProperties = {
  background: '#fff',
  padding: '8px 6px',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
};
