import { useState, useCallback, useEffect } from 'react';
import { Spin, Segmented, Tag, Button, Flex, Typography, message } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import { PageHeader } from '@/components/common';
import { ScheduleWeekGrid, WeekSelector } from '@/components/schedule';
import { scheduleApi, exportApi, showError } from '@/services';
import { useAppStore, useAuthStore } from '@/stores';
import type { ScheduleInfo } from '@/types';

const { Text } = Typography;

type ViewMode = 'all' | 'mine';

export default function SchedulePage() {
  const { currentSemester } = useAppStore();
  const { isAdmin, isLeader } = useAuthStore();
  const semesterId = currentSemester?.id;

  const canManage = isAdmin() || isLeader();
  const defaultView: ViewMode = canManage ? 'all' : 'mine';

  const [viewMode, setViewMode] = useState<ViewMode>(defaultView);
  const [weekNumber, setWeekNumber] = useState(1);
  const [loading, setLoading] = useState(false);
  const [schedule, setSchedule] = useState<ScheduleInfo | null>(null);
  const [exportLoading, setExportLoading] = useState(false);

  const fetchSchedule = useCallback(async () => {
    setLoading(true);
    try {
      const api = viewMode === 'mine'
        ? scheduleApi.getMySchedule
        : scheduleApi.getSchedule;
      const { data } = await api(semesterId);
      setSchedule(data.data);
    } catch (err) {
      showError(err, '加载排班数据失败');
      setSchedule(null);
    } finally {
      setLoading(false);
    }
  }, [semesterId, viewMode]);

  useEffect(() => {
    fetchSchedule();
  }, [fetchSchedule]);

  const handleExport = async () => {
    if (!semesterId) {
      message.warning('请先选择学期');
      return;
    }
    setExportLoading(true);
    try {
      const response = await exportApi.exportSchedule(semesterId);
      // Blob 下载
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `排班表_${currentSemester?.name || ''}.xlsx`);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      message.success('导出成功');
    } catch (err) {
      showError(err, '导出失败');
    } finally {
      setExportLoading(false);
    }
  };

  const isPublished = schedule?.status === 'published';
  const items = schedule?.items || [];

  return (
    <div>
      <PageHeader
        title="排班概览"
        description={currentSemester ? `当前学期：${currentSemester.name}` : '未设置当前学期'}
        extra={
          <Flex gap={12} align="center">
            <WeekSelector weekNumber={weekNumber} onChange={setWeekNumber} />
            {canManage && (
              <Button
                icon={<DownloadOutlined />}
                loading={exportLoading}
                onClick={handleExport}
              >
                导出Excel
              </Button>
            )}
          </Flex>
        }
      />

      {/* 视图切换 */}
      <Flex justify="space-between" align="center" style={{ marginBottom: 16 }}>
        <Segmented
          options={[
            { label: '全部排班', value: 'all' },
            { label: '仅我的', value: 'mine' },
          ]}
          value={viewMode}
          onChange={(v) => setViewMode(v as ViewMode)}
        />
        <Flex gap={8} align="center">
          {isPublished ? (
            <Tag color="success">已发布</Tag>
          ) : schedule ? (
            <Tag color="warning">草稿</Tag>
          ) : null}
          {schedule && items.length > 0 && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              共 {items.length} 条排班记录
            </Text>
          )}
        </Flex>
      </Flex>

      <Spin spinning={loading}>
        <ScheduleWeekGrid
          items={items}
          weekNumber={weekNumber}
        />
      </Spin>
    </div>
  );
}
