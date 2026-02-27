import { useState, useEffect, useCallback } from 'react';
import { Select, message, Typography, Flex } from 'antd';
import WeekGridView from '@/components/config/WeekGridView';
import { semesterApi, timeSlotApi } from '@/services/configApi';
import type { SemesterInfo, TimeSlotInfo } from '@/types';

const { Text } = Typography;

export default function TimeSlotTab() {
  const [semesters, setSemesters] = useState<SemesterInfo[]>([]);
  const [selectedSemesterId, setSelectedSemesterId] = useState<string>('');
  const [timeSlots, setTimeSlots] = useState<TimeSlotInfo[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchSemesters = useCallback(async () => {
    try {
      const { data } = await semesterApi.list();
      const list = data.data as SemesterInfo[];
      setSemesters(list);
      const active = list.find((s) => s.is_active);
      if (active) setSelectedSemesterId(active.id);
      else if (list.length > 0) setSelectedSemesterId(list[0].id);
    } catch {
      /* 静默 */
    }
  }, []);

  const fetchTimeSlots = useCallback(async () => {
    if (!selectedSemesterId) return;
    setLoading(true);
    try {
      const { data } = await timeSlotApi.list({ semester_id: selectedSemesterId });
      setTimeSlots(data.data as TimeSlotInfo[]);
    } catch {
      message.error('获取时间段列表失败');
    } finally {
      setLoading(false);
    }
  }, [selectedSemesterId]);

  useEffect(() => { fetchSemesters(); }, [fetchSemesters]);
  useEffect(() => { fetchTimeSlots(); }, [fetchTimeSlots]);

  const currentSemester = semesters.find((s) => s.id === selectedSemesterId);

  return (
    <div>
      <Flex gap={12} align="center" style={{ marginBottom: 16 }}>
        <Text type="secondary">选择学期：</Text>
        <Select
          style={{ width: 260 }}
          value={selectedSemesterId || undefined}
          onChange={setSelectedSemesterId}
          options={semesters.map((s) => ({
            label: s.name + (s.is_active ? ' (当前)' : ''),
            value: s.id,
          }))}
          placeholder="请选择学期"
        />
      </Flex>
      <WeekGridView
        timeSlots={timeSlots}
        loading={loading}
        semester={currentSemester}
        onRefresh={fetchTimeSlots}
      />
    </div>
  );
}
