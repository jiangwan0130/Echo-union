import { useState, useEffect, useCallback } from 'react';
import { Button, Space, Tag, message, Empty, Spin, Alert } from 'antd';
import {
  SendOutlined,
  HistoryOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { PageHeader } from '@/components/common';
import {
  WeekSelector,
  ScheduleWeekGrid,
  CandidateDrawer,
  PublishModal,
  ChangeLogDrawer,
} from '@/components/schedule';
import { scheduleApi, locationApi, showError } from '@/services';
import { useAppStore } from '@/stores';
import type { ScheduleInfo, ScheduleItem, TimeSlotBrief, LocationInfo, ScopeCheckResponse } from '@/types';

export default function AdjustSchedulePage() {
  const { currentSemester } = useAppStore();
  const navigate = useNavigate();
  const semesterId = currentSemester?.id;

  const [schedule, setSchedule] = useState<ScheduleInfo | null>(null);
  const [locations, setLocations] = useState<LocationInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [weekNumber, setWeekNumber] = useState(1);

  // Drawer / Modal 状态
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedSlot, setSelectedSlot] = useState<TimeSlotBrief | null>(null);
  const [selectedItems, setSelectedItems] = useState<ScheduleItem[]>([]);

  const [publishOpen, setPublishOpen] = useState(false);
  const [changeLogOpen, setChangeLogOpen] = useState(false);

  // Scope check 状态
  const [scopeAlert, setScopeAlert] = useState<ScopeCheckResponse | null>(null);
  const [scopeAlertDismissed, setScopeAlertDismissed] = useState(false);

  const isPublished = schedule?.status === 'published';

  // 加载排班
  const fetchSchedule = useCallback(async () => {
    if (!semesterId) return;
    setLoading(true);
    try {
      const { data } = await scheduleApi.getSchedule(semesterId);
      setSchedule(data.data);
    } catch (err) {
      showError(err, '加载排班数据失败');
    } finally {
      setLoading(false);
    }
  }, [semesterId]);

  // 加载地点
  const fetchLocations = useCallback(async () => {
    try {
      const { data } = await locationApi.list();
      setLocations(data.data || []);
    } catch {
      // 非关键，静默
    }
  }, []);

  useEffect(() => {
    fetchSchedule();
    fetchLocations();
  }, [fetchSchedule, fetchLocations]);

  // Scope check: 排班加载后检测人员范围变更
  useEffect(() => {
    if (!schedule?.id) {
      setScopeAlert(null);
      return;
    }
    let cancelled = false;
    scheduleApi
      .checkScope(schedule.id)
      .then(({ data }) => {
        if (!cancelled && data.data.changed) {
          setScopeAlert(data.data);
          setScopeAlertDismissed(false);
        } else if (!cancelled) {
          setScopeAlert(null);
        }
      })
      .catch(() => {
        // 非关键，静默
      });
    return () => {
      cancelled = true;
    };
  }, [schedule?.id]);

  // 时段卡片点击 → 打开候选人抽屉
  const handleSlotClick = (timeSlot: TimeSlotBrief, slotItems: ScheduleItem[]) => {
    setSelectedSlot(timeSlot);
    setSelectedItems(slotItems);
    setDrawerOpen(true);
  };

  // 候选人抽屉保存后刷新
  const handleCandidateSaved = () => {
    setDrawerOpen(false);
    fetchSchedule();
  };

  // 发布 / 修改确认
  const handlePublishConfirm = async () => {
    try {
      await scheduleApi.publish();
      message.success('排班表已发布');
      setPublishOpen(false);
      fetchSchedule();
    } catch (err) {
      showError(err, '发布失败');
      throw err; // 让 PublishModal 感知失败
    }
  };

  return (
    <div>
      <PageHeader
        title="手动调整排班"
        description={
          currentSemester
            ? `当前学期：${currentSemester.name}`
            : '未设置当前学期'
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchSchedule}>
              刷新
            </Button>
            {schedule && (
              <Button
                icon={<HistoryOutlined />}
                onClick={() => setChangeLogOpen(true)}
              >
                变更记录
              </Button>
            )}
            {schedule && (
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={() => setPublishOpen(true)}
              >
                {isPublished ? '已发布' : '发布排班'}
              </Button>
            )}
          </Space>
        }
      />

      {/* 人员范围变更警告 */}
      {scopeAlert?.changed && !scopeAlertDismissed && (
        <Alert
          type="warning"
          showIcon
          closable
          onClose={() => setScopeAlertDismissed(true)}
          style={{ marginBottom: 16 }}
          message="值班人员范围已变更"
          description={
            <Space direction="vertical" size={4}>
              <span>
                {scopeAlert.added_users?.length
                  ? `新增: ${scopeAlert.added_users.join(', ')}`
                  : ''}
                {scopeAlert.added_users?.length && scopeAlert.removed_users?.length
                  ? '；'
                  : ''}
                {scopeAlert.removed_users?.length
                  ? `移除: ${scopeAlert.removed_users.length} 人`
                  : ''}
              </span>
              <Space>
                <Button
                  size="small"
                  type="primary"
                  onClick={() => navigate('/schedule/auto')}
                >
                  前往自动排班
                </Button>
              </Space>
            </Space>
          }
        />
      )}

      {/* 状态标签 + 周选择器 */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Space>
          {schedule && (
            <Tag color={isPublished ? 'green' : 'orange'}>
              {isPublished ? '已发布' : '草稿'}
            </Tag>
          )}
        </Space>
        <WeekSelector weekNumber={weekNumber} onChange={setWeekNumber} />
      </div>

      {/* 排班网格 */}
      <Spin spinning={loading}>
        {schedule?.items ? (
          <ScheduleWeekGrid
            items={schedule.items}
            weekNumber={weekNumber}
            editable
            onSlotClick={handleSlotClick}
          />
        ) : (
          !loading && (
            <Empty
              description="暂无排班数据，请先执行自动排班"
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          )
        )}
      </Spin>

      {/* 候选人抽屉 */}
      <CandidateDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        timeSlot={selectedSlot}
        currentItems={selectedItems}
        locations={locations}
        isPublished={isPublished}
        onSaved={handleCandidateSaved}
      />

      {/* 发布弹窗 */}
      <PublishModal
        open={publishOpen}
        mode={isPublished ? 'modify' : 'publish'}
        onCancel={() => setPublishOpen(false)}
        onConfirm={handlePublishConfirm}
      />

      {/* 变更记录抽屉 */}
      {schedule && (
        <ChangeLogDrawer
          open={changeLogOpen}
          onClose={() => setChangeLogOpen(false)}
          scheduleId={schedule.id}
        />
      )}
    </div>
  );
}
