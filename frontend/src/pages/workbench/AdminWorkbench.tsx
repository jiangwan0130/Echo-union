import { useState, useEffect, useCallback } from 'react';
import {
  Steps,
  Button,
  Space,
  Alert,
  Card,
  message,
  Spin,
  Result,
  Modal,
  Tabs,
  Tooltip,
} from 'antd';
import {
  SettingOutlined,
  TeamOutlined,
  ScheduleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useAppStore } from '@/stores';
import { semesterApi } from '@/services';
import type { PhaseCheckResponse, SemesterPhase } from '@/types';

// 复用现有配置子组件
import TimeSlotTab from '@/pages/admin/config/TimeSlotTab';
import LocationTab from '@/pages/admin/config/LocationTab';
import ScheduleRuleTab from '@/pages/admin/config/ScheduleRuleTab';

// 复用现有排班/进度组件（懒加载避免循环依赖）
import { lazy, Suspense } from 'react';
const ProgressPage = lazy(() => import('@/pages/admin/progress/ProgressPage'));
const AutoSchedulePage = lazy(() => import('@/pages/schedule/AutoSchedulePage'));
const AdjustSchedulePage = lazy(() => import('@/pages/schedule/AdjustSchedulePage'));
const SchedulePage = lazy(() => import('@/pages/schedule/SchedulePage'));

// 阶段人员选择
import DutyMemberSelector from './DutyMemberSelector';

const PHASE_ORDER: SemesterPhase[] = ['configuring', 'collecting', 'scheduling', 'published'];

const PHASE_STEP_MAP: Record<SemesterPhase, number> = {
  configuring: 0,
  collecting: 1,
  scheduling: 2,
  published: 3,
};

const stepItems = [
  { title: '系统配置', icon: <SettingOutlined /> },
  { title: '收集时间表', icon: <TeamOutlined /> },
  { title: '排班', icon: <ScheduleOutlined /> },
  { title: '排班结果', icon: <CheckCircleOutlined /> },
];

export default function AdminWorkbench() {
  const { currentSemester, fetchCurrentSemester } = useAppStore();
  const [phaseCheck, setPhaseCheck] = useState<PhaseCheckResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [advancing, setAdvancing] = useState(false);

  const phase = currentSemester?.phase ?? 'configuring';
  const semesterId = currentSemester?.id;
  const currentStep = PHASE_STEP_MAP[phase] ?? 0;

  const loadPhaseCheck = useCallback(async () => {
    if (!semesterId) return;
    try {
      const { data } = await semesterApi.checkPhase(semesterId);
      setPhaseCheck(data.data);
    } catch {
      // ignore
    }
  }, [semesterId]);

  useEffect(() => {
    loadPhaseCheck();
  }, [loadPhaseCheck]);

  const handleAdvance = async () => {
    if (!semesterId) return;
    const nextPhase = PHASE_ORDER[currentStep + 1];
    if (!nextPhase) return;

    // 从 collecting 回退或前进需要确认
    if (nextPhase === 'collecting') {
      Modal.confirm({
        title: '激活排班',
        icon: <ExclamationCircleOutlined />,
        content: '激活后，所有选定的值班人员将看到提交时间表的通知。确认继续？',
        onOk: () => doAdvance(nextPhase),
      });
      return;
    }

    await doAdvance(nextPhase);
  };

  const doAdvance = async (targetPhase: SemesterPhase) => {
    if (!semesterId) return;
    setAdvancing(true);
    try {
      await semesterApi.advancePhase(semesterId, { target_phase: targetPhase });
      message.success('阶段推进成功');
      await fetchCurrentSemester();
      await loadPhaseCheck();
    } catch {
      message.error('阶段推进失败，请检查前置条件');
    } finally {
      setAdvancing(false);
    }
  };

  const handleGoBack = async () => {
    if (!semesterId || currentStep === 0) return;
    const prevPhase = PHASE_ORDER[currentStep - 1];

    Modal.confirm({
      title: '回退阶段',
      icon: <ExclamationCircleOutlined />,
      content: '回退不会清除已有数据（已提交的时间表、已配置的内容保留）。确认回退？',
      onOk: async () => {
        setAdvancing(true);
        try {
          await semesterApi.advancePhase(semesterId, { target_phase: prevPhase });
          message.success('已回退');
          await fetchCurrentSemester();
          await loadPhaseCheck();
        } catch {
          message.error('回退失败');
        } finally {
          setAdvancing(false);
        }
      },
    });
  };

  if (!currentSemester) {
    return (
      <Result
        status="info"
        title="暂无活跃学期"
        subTitle="请先在用户管理中创建并激活一个学期"
      />
    );
  }

  return (
    <div>
      <Card style={{ marginBottom: 24 }}>
        <div style={{ marginBottom: 8, fontSize: 16, fontWeight: 500 }}>
          当前学期：{currentSemester.name}
        </div>
        <Steps
          current={currentStep}
          items={stepItems}
          style={{ marginBottom: 0 }}
        />
      </Card>

      {/* Step 内容区 */}
      <Card>
        {phase === 'configuring' && (
          <ConfigStep
            onRefreshCheck={loadPhaseCheck}
          />
        )}

        {phase === 'collecting' && (
          <CollectingStep semesterId={semesterId!} />
        )}

        {phase === 'scheduling' && (
          <SchedulingStep />
        )}

        {phase === 'published' && (
          <PublishedStep />
        )}
      </Card>

      {/* 底部操作栏 */}
      <Card style={{ marginTop: 16 }}>
        <Space style={{ display: 'flex', justifyContent: 'space-between' }}>
          <div>
            {currentStep > 0 && phase !== 'published' && (
              <Button onClick={handleGoBack} loading={advancing}>
                上一步
              </Button>
            )}
          </div>
          <div>
            {phase !== 'published' && (
              <Tooltip
                title={
                  phaseCheck && !phaseCheck.can_advance
                    ? phaseCheck.checks
                        .filter((c) => !c.passed)
                        .map((c) => c.message || c.label)
                        .join('；')
                    : ''
                }
              >
                <Button
                  type="primary"
                  onClick={handleAdvance}
                  loading={advancing}
                  disabled={!phaseCheck?.can_advance}
                >
                  {currentStep === 0 && '下一步：激活排班'}
                  {currentStep === 1 && '下一步：开始排班'}
                  {currentStep === 2 && '下一步：预览结果'}
                </Button>
              </Tooltip>
            )}
          </div>
        </Space>
        {/* 展示未满足条件 */}
        {phaseCheck && !phaseCheck.can_advance && (
          <div style={{ marginTop: 12 }}>
            {phaseCheck.checks
              .filter((c) => !c.passed)
              .map((c, i) => (
                <Alert
                  key={i}
                  type="warning"
                  message={c.label}
                  description={c.message}
                  showIcon
                  style={{ marginBottom: 8 }}
                />
              ))}
          </div>
        )}
      </Card>
    </div>
  );
}

// ── Step 1: 系统配置 ──

function ConfigStep({ onRefreshCheck }: { onRefreshCheck: () => void }) {
  const handleTabChange = () => {
    // 每次切 tab 刷新检查状态
    setTimeout(onRefreshCheck, 500);
  };

  return (
    <div>
      <Alert
        type="info"
        message="请完成以下配置后进入下一步"
        description="配置时间段、地点和排班规则。至少需要1个时间段和1个地点。"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <Tabs
        onChange={handleTabChange}
        items={[
          {
            key: 'timeslot',
            label: '时间段配置',
            children: <TimeSlotTab />,
          },
          {
            key: 'location',
            label: '值班地点',
            children: <LocationTab />,
          },
          {
            key: 'rule',
            label: '排班规则',
            children: <ScheduleRuleTab />,
          },
          {
            key: 'members',
            label: '选定值班人员',
            children: <DutyMemberSelector />,
          },
        ]}
      />
    </div>
  );
}

// ── Step 2: 收集时间表 ──

function CollectingStep({ semesterId }: { semesterId: string }) {
  return (
    <div>
      <Alert
        type="info"
        message="等待成员提交时间表"
        description="所有值班人员已收到提交时间表的通知。全员提交后可进入排班步骤。"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <Suspense fallback={<Spin />}>
        <ProgressPage />
      </Suspense>
    </div>
  );
}

// ── Step 3: 排班 ──

function SchedulingStep() {
  const [subView, setSubView] = useState<'auto' | 'adjust'>('auto');

  return (
    <div>
      <Tabs
        activeKey={subView}
        onChange={(key) => setSubView(key as 'auto' | 'adjust')}
        items={[
          {
            key: 'auto',
            label: '自动排班',
            children: (
              <Suspense fallback={<Spin />}>
                <AutoSchedulePage />
              </Suspense>
            ),
          },
          {
            key: 'adjust',
            label: '手动调整',
            children: (
              <Suspense fallback={<Spin />}>
                <AdjustSchedulePage />
              </Suspense>
            ),
          },
        ]}
      />
    </div>
  );
}

// ── Step 4: 排班结果 ──

function PublishedStep() {
  return (
    <div>
      <Result
        status="success"
        title="排班已发布"
        subTitle="所有成员可在工作台查看自己的排班安排"
      />
      <Suspense fallback={<Spin />}>
        <SchedulePage />
      </Suspense>
    </div>
  );
}
