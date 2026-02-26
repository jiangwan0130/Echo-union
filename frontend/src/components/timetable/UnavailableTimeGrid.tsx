import { useMemo } from 'react';
import { Typography, Tooltip, Empty } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import type { CourseInfo, UnavailableTime } from '@/types';

const { Text } = Typography;

const WEEKDAYS = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
];

/** 时间小时列表 (8:00 ~ 20:00) */
const HOURS = Array.from({ length: 13 }, (_, i) => i + 8);

/** 规范化时间 → 小时数 (如 "08:30:00" → 8.5) */
function timeToHour(t: string): number {
  const parts = t.replace(/:\d{2}$/, '').split(':');
  return parseInt(parts[0], 10) + parseInt(parts[1], 10) / 60;
}

/** 格式化时间 "08:10:00" → "08:10" */
const fmt = (t: string) => t.replace(/:\d{2}$/, '');

interface UnavailableTimeGridProps {
  courses: CourseInfo[];
  unavailableTimes: UnavailableTime[];
  /** 是否处于禁用状态（未导入课表时） */
  disabled?: boolean;
  /** 点击空白格回调 */
  onCellClick?: (dayOfWeek: number, hour: number) => void;
  /** 点击不可用时间色块回调 */
  onUnavailableClick?: (item: UnavailableTime) => void;
}

/**
 * 时间表周视图（只读课程底色 + 可交互不可用时间标记）
 * 基于 WeekGridView 的视觉风格独立实现，不修改原组件。
 */
export default function UnavailableTimeGrid({
  courses,
  unavailableTimes,
  disabled = false,
  onCellClick,
  onUnavailableClick,
}: UnavailableTimeGridProps) {
  // 构建课程映射: dayOfWeek + hour → CourseInfo[]
  const courseMap = useMemo(() => {
    const map = new Map<string, CourseInfo[]>();
    courses.forEach((c) => {
      const startH = Math.floor(timeToHour(c.start_time));
      const endH = Math.ceil(timeToHour(c.end_time));
      for (let h = startH; h < endH; h++) {
        const key = `${c.day_of_week}_${h}`;
        const arr = map.get(key) || [];
        arr.push(c);
        map.set(key, arr);
      }
    });
    return map;
  }, [courses]);

  // 构建不可用时间映射: dayOfWeek + hour → UnavailableTime[]
  const unavailableMap = useMemo(() => {
    const map = new Map<string, UnavailableTime[]>();
    unavailableTimes.forEach((u) => {
      const startH = Math.floor(timeToHour(u.start_time));
      const endH = Math.ceil(timeToHour(u.end_time));
      for (let h = startH; h < endH; h++) {
        const key = `${u.day_of_week}_${h}`;
        const arr = map.get(key) || [];
        arr.push(u);
        map.set(key, arr);
      }
    });
    return map;
  }, [unavailableTimes]);

  if (disabled) {
    return (
      <div
        style={{
          position: 'relative',
          borderRadius: 8,
          overflow: 'hidden',
          border: '1px solid #f0f0f0',
        }}
      >
        {renderGrid(true)}
        {/* 禁用遮罩 */}
        <div
          style={{
            position: 'absolute',
            inset: 0,
            background: 'rgba(255,255,255,0.75)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 10,
            backdropFilter: 'blur(1px)',
          }}
        >
          <Empty
            description="请先导入课程表"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        </div>
      </div>
    );
  }

  return (
    <div style={{ borderRadius: 8, overflow: 'hidden', border: '1px solid #f0f0f0' }}>
      {renderGrid(false)}
      <style>{`
        .timetable-cell:hover {
          background: #f0f5ff !important;
        }
        .unavailable-block:hover {
          opacity: 0.85;
          box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
      `}</style>
    </div>
  );

  function renderGrid(isDisabled: boolean) {
    return (
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '70px repeat(5, 1fr)',
          gap: 1,
          background: '#f0f0f0',
        }}
      >
        {/* 表头 */}
        <div style={headerStyle}>时间</div>
        {WEEKDAYS.map((day) => (
          <div key={day.value} style={headerStyle}>
            {day.label}
          </div>
        ))}

        {/* 每小时行 */}
        {HOURS.map((hour) => [
          <div key={`h-${hour}`} style={timeCellStyle}>
            <Text style={{ fontSize: 12, fontWeight: 500 }}>
              {String(hour).padStart(2, '0')}:00
            </Text>
          </div>,
          ...WEEKDAYS.map((day) => {
            const key = `${day.value}_${hour}`;
            const coursesInCell = courseMap.get(key) || [];
            const unavailableInCell = unavailableMap.get(key) || [];
            const hasCourse = coursesInCell.length > 0;
            const hasUnavailable = unavailableInCell.length > 0;

            return (
              <div
                key={key}
                className={!isDisabled && !hasCourse && !hasUnavailable ? 'timetable-cell' : ''}
                style={{
                  background: hasCourse ? '#e6f4ff' : '#fff',
                  padding: 4,
                  minHeight: 48,
                  cursor: isDisabled ? 'default' : 'pointer',
                  position: 'relative',
                }}
                onClick={() => {
                  if (isDisabled) return;
                  if (hasUnavailable) {
                    onUnavailableClick?.(unavailableInCell[0]);
                  } else if (!hasCourse) {
                    onCellClick?.(day.value, hour);
                  }
                }}
              >
                {/* 课程展示 */}
                {hasCourse && (
                  <Tooltip title={coursesInCell.map((c) => `${c.name} ${fmt(c.start_time)}-${fmt(c.end_time)}`).join('\n')}>
                    <div style={courseBlockStyle}>
                      <Text style={{ fontSize: 11, color: '#1677ff' }} ellipsis>
                        📘 {coursesInCell[0].name}
                      </Text>
                    </div>
                  </Tooltip>
                )}

                {/* 不可用时间展示 */}
                {hasUnavailable && (
                  <Tooltip
                    title={unavailableInCell.map((u) =>
                      `${fmt(u.start_time)}-${fmt(u.end_time)} ${u.reason || '不可用'}`
                    ).join('\n')}
                  >
                    <div className="unavailable-block" style={unavailableBlockStyle}>
                      <Text style={{ fontSize: 11, color: '#fff' }} ellipsis>
                        🚫 {unavailableInCell[0].reason || '不可用'}
                      </Text>
                    </div>
                  </Tooltip>
                )}

                {/* 空白格 + 提示 */}
                {!hasCourse && !hasUnavailable && !isDisabled && (
                  <div style={emptyHintStyle}>
                    <PlusOutlined style={{ color: '#d9d9d9', fontSize: 12 }} />
                  </div>
                )}
              </div>
            );
          }),
        ])}
      </div>
    );
  }
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
  padding: '6px 4px',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
};

const courseBlockStyle: React.CSSProperties = {
  background: '#bae0ff',
  borderRadius: 4,
  padding: '2px 6px',
  marginBottom: 2,
};

const unavailableBlockStyle: React.CSSProperties = {
  background: '#ff4d4f',
  borderRadius: 4,
  padding: '2px 6px',
  cursor: 'pointer',
  transition: 'all 0.2s',
};

const emptyHintStyle: React.CSSProperties = {
  width: '100%',
  height: '100%',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  opacity: 0,
  transition: 'opacity 0.2s',
};
