import { useState, useEffect, useCallback } from 'react';
import {
  Drawer,
  List,
  Checkbox,
  Tag,
  Space,
  Button,
  Select,
  Typography,
  Spin,
  message,
  Input,
} from 'antd';
import {
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { scheduleApi, showError } from '@/services';
import type {
  ScheduleItem,
  TimeSlotBrief,
  CandidateInfo,
  LocationInfo,
} from '@/types';

const { Text, Title } = Typography;

/** 规范化时间 "08:10:00" → "08:10" */
const fmt = (t: string) => t.replace(/:\d{2}$/, '');

const WEEKDAY_LABELS: Record<number, string> = {
  1: '周一', 2: '周二', 3: '周三', 4: '周四', 5: '周五',
};

interface CandidateDrawerProps {
  open: boolean;
  onClose: () => void;
  timeSlot: TimeSlotBrief | null;
  currentItems: ScheduleItem[];
  locations: LocationInfo[];
  /** 排班表是否已发布 */
  isPublished: boolean;
  onSaved: () => void;
}

/**
 * 候选人侧边面板
 * 展示可选人员列表 + 冲突提示 + 保存修改
 */
export default function CandidateDrawer({
  open,
  onClose,
  timeSlot,
  currentItems,
  locations,
  isPublished,
  onSaved,
}: CandidateDrawerProps) {
  const [candidates, setCandidates] = useState<CandidateInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [locationId, setLocationId] = useState<string>('');
  const [saveLoading, setSaveLoading] = useState(false);
  const [reason, setReason] = useState('');

  const itemId = currentItems[0]?.id;

  // 加载候选人
  const fetchCandidates = useCallback(async () => {
    if (!itemId) return;
    setLoading(true);
    try {
      const { data } = await scheduleApi.getCandidates(itemId);
      setCandidates(data.data || []);
    } catch (err) {
      showError(err, '加载候选人失败');
    } finally {
      setLoading(false);
    }
  }, [itemId]);

  useEffect(() => {
    if (open && itemId) {
      fetchCandidates();
      // 初始化已选状态
      const currentMemberIds = new Set(
        currentItems.map((i) => i.member?.id).filter(Boolean) as string[],
      );
      setSelectedIds(currentMemberIds);
      setLocationId(currentItems[0]?.location?.id || '');
      setReason('');
    }
  }, [open, itemId, currentItems, fetchCandidates]);

  const toggleCandidate = (userId: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(userId)) {
        next.delete(userId);
      } else {
        next.add(userId);
      }
      return next;
    });
  };

  const handleSave = async () => {
    if (!itemId) return;

    if (isPublished && !reason.trim()) {
      message.warning('发布后修改需要填写修改原因');
      return;
    }

    setSaveLoading(true);
    try {
      // 对每个选中的候选人逐个更新（API 设计为单条更新）
      // 如果只有一个 item 对应一个时段，更新该 item 的 member
      if (isPublished) {
        for (const memberId of selectedIds) {
          await scheduleApi.updatePublishedItem(itemId, {
            member_id: memberId,
            reason: reason.trim(),
          });
        }
      } else {
        for (const memberId of selectedIds) {
          await scheduleApi.updateItem(itemId, {
            member_id: memberId,
            location_id: locationId || undefined,
          });
        }
      }
      message.success('排班已更新');
      onSaved();
      onClose();
    } catch (err) {
      showError(err, '保存失败');
    } finally {
      setSaveLoading(false);
    }
  };

  return (
    <Drawer
      title="编辑排班"
      open={open}
      onClose={onClose}
      width={420}
      footer={
        <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
          <Button onClick={onClose}>取消</Button>
          <Button type="primary" loading={saveLoading} onClick={handleSave}>
            保存
          </Button>
        </Space>
      }
    >
      {/* 时段信息 */}
      {timeSlot && (
        <div style={{ marginBottom: 16 }}>
          <Title level={5} style={{ margin: 0 }}>
            {WEEKDAY_LABELS[timeSlot.day_of_week]} {fmt(timeSlot.start_time)}-
            {fmt(timeSlot.end_time)}
          </Title>
          <Text type="secondary">{timeSlot.name}</Text>
        </div>
      )}

      {/* 地点选择 */}
      <div style={{ marginBottom: 16 }}>
        <Text strong style={{ display: 'block', marginBottom: 4 }}>
          值班地点
        </Text>
        <Select
          style={{ width: '100%' }}
          value={locationId || undefined}
          onChange={setLocationId}
          placeholder="选择值班地点"
          options={locations.map((l) => ({ label: l.name, value: l.id }))}
          allowClear
        />
      </div>

      {/* 已发布修改需要原因 */}
      {isPublished && (
        <div style={{ marginBottom: 16 }}>
          <Text strong style={{ display: 'block', marginBottom: 4 }}>
            修改原因 <Text type="danger">*</Text>
          </Text>
          <Input.TextArea
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="请填写修改原因"
            rows={2}
            maxLength={200}
          />
        </div>
      )}

      {/* 当前值班人员 */}
      {currentItems.length > 0 && (
        <div style={{ marginBottom: 16 }}>
          <Text strong style={{ display: 'block', marginBottom: 4 }}>
            当前值班
          </Text>
          <Space wrap>
            {currentItems.map((item) => (
              <Tag key={item.id} color="blue">
                {item.member?.name || '未分配'}
              </Tag>
            ))}
          </Space>
        </div>
      )}

      {/* 候选人列表 */}
      <Text strong style={{ display: 'block', marginBottom: 8 }}>
        候选人
      </Text>
      <Spin spinning={loading}>
        <List
          dataSource={candidates}
          locale={{ emptyText: '无可用候选人' }}
          renderItem={(candidate) => {
            const isSelected = selectedIds.has(candidate.user_id);
            const hasConflict = !candidate.available;
            return (
              <List.Item
                style={{
                  padding: '8px 12px',
                  cursor: 'pointer',
                  background: isSelected ? '#e6f4ff' : undefined,
                  borderRadius: 6,
                }}
                onClick={() => toggleCandidate(candidate.user_id)}
              >
                <Space style={{ width: '100%' }}>
                  <Checkbox checked={isSelected} />
                  <div>
                    <Text strong>{candidate.name}</Text>
                    <Text type="secondary" style={{ marginLeft: 8 }}>
                      {candidate.department?.name}
                    </Text>
                  </div>
                  {hasConflict ? (
                    <Tag
                      icon={<CloseCircleOutlined />}
                      color="error"
                      style={{ marginLeft: 'auto' }}
                    >
                      {candidate.conflicts?.[0] || '不可用'}
                    </Tag>
                  ) : candidate.available ? (
                    <Tag
                      icon={<CheckCircleOutlined />}
                      color="success"
                      style={{ marginLeft: 'auto' }}
                    >
                      可用
                    </Tag>
                  ) : (
                    <Tag icon={<WarningOutlined />} color="warning" style={{ marginLeft: 'auto' }}>
                      冲突
                    </Tag>
                  )}
                </Space>
              </List.Item>
            );
          }}
        />
      </Spin>
    </Drawer>
  );
}
