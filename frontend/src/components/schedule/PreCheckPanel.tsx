import { useEffect, useState, useCallback } from 'react';
import { Card, Flex, Typography, Spin, Tag } from 'antd';
import {
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import {
  timetableApi,
  scheduleRuleApi,
  scheduleApi,
  showError,
} from '@/services';
import type { ScheduleRuleInfo, TimetableProgressResponse, ScopeCheckResponse } from '@/types';

const { Text } = Typography;

interface CheckItem {
  label: string;
  status: 'pass' | 'warn' | 'fail';
  detail?: string;
}

interface PreCheckPanelProps {
  semesterId?: string;
  scheduleId?: string;
  onAllPassed: (passed: boolean) => void;
}

/**
 * 自动排班前置检查面板
 * 检查：学期/时间段/值班人员/时间表提交率/排班规则/值班地点
 */
export default function PreCheckPanel({
  semesterId,
  scheduleId,
  onAllPassed,
}: PreCheckPanelProps) {
  const [loading, setLoading] = useState(false);
  const [checks, setChecks] = useState<CheckItem[]>([]);
  const [rules, setRules] = useState<ScheduleRuleInfo[]>([]);

  const runChecks = useCallback(async () => {
    if (!semesterId) {
      setChecks([
        { label: '学期已配置', status: 'fail', detail: '未选择学期' },
      ]);
      onAllPassed(false);
      return;
    }

    setLoading(true);
    try {
      const results: CheckItem[] = [];

      // 学期
      results.push({ label: '学期已配置', status: 'pass' });

      // 并行加载
      const promises: Promise<unknown>[] = [
        timetableApi.getProgress(),
        scheduleRuleApi.list(),
      ];
      if (scheduleId) {
        promises.push(scheduleApi.checkScope(scheduleId));
      }
      const settled = await Promise.allSettled(promises);
      const [progressRes, rulesRes] = settled;
      const scopeRes = scheduleId ? settled[2] : undefined;

      // 时间表提交进度
      if (progressRes.status === 'fulfilled') {
        const progress = (progressRes.value as { data: { data: TimetableProgressResponse } }).data.data;
        if (progress.total === 0) {
          results.push({
            label: '值班人员已确定',
            status: 'fail',
            detail: '暂无值班人员',
          });
        } else {
          results.push({
            label: `值班人员已确定 (${progress.total}人)`,
            status: 'pass',
          });
        }

        if (progress.progress >= 100) {
          results.push({
            label: `时间表提交率: ${progress.progress}% (${progress.submitted}/${progress.total})`,
            status: 'pass',
          });
        } else {
          results.push({
            label: `时间表提交率: ${progress.progress}% (${progress.submitted}/${progress.total})`,
            status: 'fail',
            detail: `${progress.total - progress.submitted}人未提交，无法排班`,
          });
        }
      } else {
        results.push({
          label: '时间表提交进度',
          status: 'warn',
          detail: '无法获取',
        });
      }

      // 排班规则
      if (rulesRes.status === 'fulfilled') {
        const ruleList = (rulesRes.value as { data: { data: ScheduleRuleInfo[] } }).data.data;
        setRules(ruleList);
        results.push({
          label: `排班规则已配置 (${ruleList.filter((r) => r.is_enabled).length}条启用)`,
          status: 'pass',
        });
      } else {
        results.push({
          label: '排班规则',
          status: 'warn',
          detail: '无法获取',
        });
      }

      // 人员范围检测
      if (scopeRes) {
        if (scopeRes.status === 'fulfilled') {
          const scope = (
            scopeRes as PromiseFulfilledResult<{
              data: { data: ScopeCheckResponse };
            }>
          ).value.data.data;
          if (scope.changed) {
            const parts: string[] = [];
            if (scope.added_users?.length)
              parts.push(`新增 ${scope.added_users.length} 人`);
            if (scope.removed_users?.length)
              parts.push(`移除 ${scope.removed_users.length} 人`);
            results.push({
              label: `人员范围检测: ${parts.join(', ')}`,
              status: 'warn',
              detail: '人员范围已变更，建议重新排班',
            });
          } else {
            results.push({
              label: '人员范围检测: 无变更',
              status: 'pass',
            });
          }
        } else {
          results.push({
            label: '人员范围检测',
            status: 'warn',
            detail: '无法检测',
          });
        }
      }

      setChecks(results);
      const allPassed = results.every((r) => r.status !== 'fail');
      onAllPassed(allPassed);
    } catch (err) {
      showError(err, '前置检查失败');
      onAllPassed(false);
    } finally {
      setLoading(false);
    }
  }, [semesterId, scheduleId, onAllPassed]);

  useEffect(() => {
    runChecks();
  }, [runChecks]);

  const statusIcon = (status: CheckItem['status']) => {
    switch (status) {
      case 'pass':
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
      case 'warn':
        return <WarningOutlined style={{ color: '#faad14' }} />;
      case 'fail':
        return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
    }
  };

  return (
    <Card title="前置检查" size="small">
      <Spin spinning={loading}>
        <Flex wrap="wrap" gap={12}>
          {checks.map((check, idx) => (
            <div key={idx} style={{ minWidth: 240 }}>
              <Flex gap={8} align="center">
                {statusIcon(check.status)}
                <Text>{check.label}</Text>
              </Flex>
              {check.detail && (
                <Text
                  type={check.status === 'fail' ? 'danger' : 'warning'}
                  style={{ fontSize: 12, marginLeft: 24 }}
                >
                  {check.detail}
                </Text>
              )}
            </div>
          ))}
        </Flex>

        {/* 规则详情 */}
        {rules.length > 0 && (
          <div style={{ marginTop: 12 }}>
            <Text strong style={{ fontSize: 12 }}>
              当前规则：
            </Text>
            <Flex gap={4} wrap="wrap" style={{ marginTop: 4 }}>
              {rules.map((rule) => (
                <Tag
                  key={rule.id}
                  color={rule.is_enabled ? 'blue' : 'default'}
                >
                  {rule.is_enabled ? '☑' : '☐'} {rule.rule_name}
                </Tag>
              ))}
            </Flex>
          </div>
        )}
      </Spin>
    </Card>
  );
}
