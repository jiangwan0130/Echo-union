import { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Switch,
  Tag,
  Tooltip,
  message,
  Typography,
  Flex,
  Empty,
} from 'antd';
import { SafetyCertificateOutlined } from '@ant-design/icons';
import { scheduleRuleApi } from '@/services/configApi';
import { showError } from '@/services/errorHandler';
import type { ScheduleRuleInfo } from '@/types';

const { Text } = Typography;

export default function ScheduleRuleTab() {
  const [rules, setRules] = useState<ScheduleRuleInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [toggling, setToggling] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await scheduleRuleApi.list();
      const raw = data.data;
      setRules(Array.isArray(raw) ? raw : (raw as unknown as { list: ScheduleRuleInfo[] }).list ?? []);
    } catch {
      message.error('获取排班规则失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleToggle = async (rule: ScheduleRuleInfo, enabled: boolean) => {
    setToggling(rule.id);
    try {
      await scheduleRuleApi.update(rule.id, { is_enabled: enabled });
      message.success(`「${rule.rule_name}」已${enabled ? '启用' : '禁用'}`);
      fetchData();
    } catch (err) {
      showError(err, '操作失败');
    } finally {
      setToggling(null);
    }
  };

  return (
    <div>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        排班规则由系统预置，可根据需要启用或禁用可配置的规则。
      </Text>
      {loading ? (
        <Table loading={loading} columns={[]} dataSource={[]} />
      ) : rules.length === 0 ? (
        <Empty description="暂无排班规则" />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {rules.map((rule) => (
            <div
              key={rule.id}
              style={{
                padding: '16px 20px',
                borderRadius: 8,
                border: '1px solid #f0f0f0',
                background: rule.is_enabled ? '#fff' : '#fafafa',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                transition: 'all 0.2s',
              }}
            >
              <div style={{ flex: 1 }}>
                <Flex align="center" gap={8}>
                  <SafetyCertificateOutlined
                    style={{ color: rule.is_enabled ? '#1677ff' : '#bbb', fontSize: 16 }}
                  />
                  <Text strong style={{ fontSize: 14 }}>
                    {rule.rule_name}
                  </Text>
                  {!rule.is_configurable && (
                    <Tag bordered={false} color="default" style={{ fontSize: 11 }}>
                      系统内置
                    </Tag>
                  )}
                </Flex>
                {rule.description && (
                  <Text type="secondary" style={{ display: 'block', marginTop: 4, marginLeft: 24 }}>
                    {rule.description}
                  </Text>
                )}
              </div>
              <Tooltip
                title={!rule.is_configurable ? '此规则为系统内置，不可修改' : undefined}
              >
                <Switch
                  checked={rule.is_enabled}
                  disabled={!rule.is_configurable}
                  loading={toggling === rule.id}
                  onChange={(checked) => handleToggle(rule, checked)}
                />
              </Tooltip>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
