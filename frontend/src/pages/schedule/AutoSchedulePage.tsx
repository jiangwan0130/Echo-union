import { useState, useEffect, useCallback } from 'react';
import { Button, Spin, Space, message } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { PageHeader } from '@/components/common';
import { PreCheckPanel, ScheduleResultPanel } from '@/components/schedule';
import { scheduleApi, showError } from '@/services';
import { useAppStore } from '@/stores';
import type { AutoScheduleResponse, ScheduleInfo } from '@/types';

export default function AutoSchedulePage() {
  const { currentSemester } = useAppStore();
  const navigate = useNavigate();
  const semesterId = currentSemester?.id;

  const [allPassed, setAllPassed] = useState(false);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<AutoScheduleResponse | null>(null);
  const [existingSchedule, setExistingSchedule] = useState<ScheduleInfo | null>(null);

  // 加载已有排班（用于 scope check）
  const fetchExistingSchedule = useCallback(async () => {
    if (!semesterId) return;
    try {
      const { data } = await scheduleApi.getSchedule(semesterId);
      setExistingSchedule(data.data);
    } catch {
      setExistingSchedule(null);
    }
  }, [semesterId]);

  useEffect(() => {
    fetchExistingSchedule();
  }, [fetchExistingSchedule]);

  const handleAutoSchedule = async () => {
    if (!semesterId) {
      message.warning('请先配置学期');
      return;
    }
    setRunning(true);
    try {
      const { data } = await scheduleApi.autoSchedule({ semester_id: semesterId });
      setResult(data.data);
      message.success('自动排班完成');
    } catch (err) {
      showError(err, '自动排班失败');
    } finally {
      setRunning(false);
    }
  };

  return (
    <div>
      <PageHeader
        title="自动排班"
        description={
          currentSemester
            ? `当前学期：${currentSemester.name}`
            : '未设置当前学期'
        }
        extra={
          <Space>
            <Button onClick={() => navigate('/schedule/adjust')}>
              前往手动调整
            </Button>
          </Space>
        }
      />

      {/* Step 1: 前置检查 */}
      <PreCheckPanel
        semesterId={semesterId}
        scheduleId={existingSchedule?.id}
        onAllPassed={setAllPassed}
      />

      {/* Step 2: 执行排班 */}
      <div style={{ marginTop: 16, textAlign: 'center', padding: '24px 0' }}>
        <Spin spinning={running}>
          <Button
            type="primary"
            size="large"
            icon={<ThunderboltOutlined />}
            disabled={!allPassed || !semesterId}
            loading={running}
            onClick={handleAutoSchedule}
          >
            开始自动排班
          </Button>
          {!allPassed && (
            <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
              请确保所有前置检查通过后再执行排班
            </div>
          )}
        </Spin>
      </div>

      {/* Step 3: 结果展示 */}
      {result && (
        <div style={{ marginTop: 16 }}>
          <ScheduleResultPanel result={result} />
        </div>
      )}
    </div>
  );
}
