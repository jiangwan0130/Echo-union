import { useState, useMemo } from 'react';
import {
  Button,
  Modal,
  Form,
  Input,
  Select,
  TimePicker,
  Tooltip,
  Typography,
  Flex,
  message,
  Empty,
  Spin,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { StatusTag } from '@/components/common';
import { timeSlotApi } from '@/services/configApi';
import { showError } from '@/services/errorHandler';
import type {
  TimeSlotInfo,
  CreateTimeSlotRequest,
  UpdateTimeSlotRequest,
  SemesterInfo,
} from '@/types';

const { Text } = Typography;

const WEEKDAYS = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
];

interface WeekGridViewProps {
  timeSlots: TimeSlotInfo[];
  loading: boolean;
  semester?: SemesterInfo;
  onRefresh: () => void;
}

export default function WeekGridView({
  timeSlots,
  loading,
  semester,
  onRefresh,
}: WeekGridViewProps) {
  const [modalOpen, setModalOpen] = useState(false);
  const [editingSlot, setEditingSlot] = useState<TimeSlotInfo | null>(null);
  const [form] = Form.useForm();
  const [submitLoading, setSubmitLoading] = useState(false);

  // 规范化时间格式: "08:10:00" → "08:10"
  const fmt = (t: string) => t.replace(/:\d{2}$/, '');

  // 按时间行分组：找出所有独立的时间区间（start_time-end_time），再按 day_of_week 分列
  const { timeRows, slotMap } = useMemo(() => {
    // 收集唯一的时间区间
    const timeRangeSet = new Map<string, { start: string; end: string }>();
    timeSlots.forEach((slot) => {
      const key = `${fmt(slot.start_time)}-${fmt(slot.end_time)}`;
      if (!timeRangeSet.has(key)) {
        timeRangeSet.set(key, { start: fmt(slot.start_time), end: fmt(slot.end_time) });
      }
    });

    // 按 start_time 排序
    const rows = [...timeRangeSet.values()].sort((a, b) =>
      a.start.localeCompare(b.start),
    );

    // 构建映射: (timeKey, dayOfWeek) -> TimeSlotInfo
    const map = new Map<string, TimeSlotInfo>();
    timeSlots.forEach((slot) => {
      map.set(`${fmt(slot.start_time)}-${fmt(slot.end_time)}_${slot.day_of_week}`, slot);
    });

    return { timeRows: rows, slotMap: map };
  }, [timeSlots]);

  const openCreate = (dayOfWeek?: number) => {
    setEditingSlot(null);
    form.resetFields();
    if (dayOfWeek) form.setFieldsValue({ day_of_week: dayOfWeek });
    setModalOpen(true);
  };

  const openEdit = (slot: TimeSlotInfo) => {
    setEditingSlot(slot);
    form.setFieldsValue({
      name: slot.name,
      day_of_week: slot.day_of_week,
      time_range: [dayjs(fmt(slot.start_time), 'HH:mm'), dayjs(fmt(slot.end_time), 'HH:mm')],
      is_active: slot.is_active,
    });
    setModalOpen(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitLoading(true);

      const startTime = values.time_range[0].format('HH:mm');
      const endTime = values.time_range[1].format('HH:mm');

      if (editingSlot) {
        const updateData: UpdateTimeSlotRequest = {
          name: values.name,
          day_of_week: values.day_of_week,
          start_time: startTime,
          end_time: endTime,
          is_active: values.is_active,
        };
        await timeSlotApi.update(editingSlot.id, updateData);
        message.success('时间段已更新');
      } else {
        const createData: CreateTimeSlotRequest = {
          name: values.name,
          semester_id: semester?.id,
          day_of_week: values.day_of_week,
          start_time: startTime,
          end_time: endTime,
        };
        await timeSlotApi.create(createData);
        message.success('时间段已创建');
      }

      setModalOpen(false);
      onRefresh();
    } catch (err) {
      showError(err, editingSlot ? '更新时间段失败' : '创建时间段失败');
    } finally {
      setSubmitLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await timeSlotApi.delete(id);
      message.success('时间段已删除');
      onRefresh();
    } catch (err) {
      showError(err, '删除失败');
    }
  };

  if (loading) {
    return (
      <Flex justify="center" align="center" style={{ padding: 80 }}>
        <Spin />
      </Flex>
    );
  }

  if (timeSlots.length === 0) {
    return (
      <div>
        <Flex justify="flex-end" style={{ marginBottom: 16 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openCreate()}>
            新建时间段
          </Button>
        </Flex>
        <Empty description="暂无时间段配置">
          <Button type="primary" onClick={() => openCreate()}>
            立即创建
          </Button>
        </Empty>
        {renderModal()}
      </div>
    );
  }

  function renderSlotCard(slot: TimeSlotInfo) {
    return (
      <div
        key={slot.id}
        style={{
          padding: '8px 10px',
          borderRadius: 8,
          border: slot.is_active ? '1px solid #e6f4ff' : '1px dashed #d9d9d9',
          background: slot.is_active ? '#e6f4ff' : '#fafafa',
          opacity: slot.is_active ? 1 : 0.6,
          transition: 'all 0.2s',
          cursor: 'pointer',
          position: 'relative',
        }}
        className="slot-card"
        onClick={() => openEdit(slot)}
      >
        <Text
          strong
          style={{
            fontSize: 13,
            display: 'block',
            marginBottom: 2,
            color: slot.is_active ? '#1677ff' : '#999',
          }}
        >
          {slot.name}
        </Text>
        <Text
          type="secondary"
          style={{ fontSize: 12 }}
        >
          {fmt(slot.start_time)}–{fmt(slot.end_time)}
        </Text>
        {!slot.is_active && (
          <div style={{ marginTop: 2 }}>
            <StatusTag active={false} />
          </div>
        )}
        <div
          style={{
            position: 'absolute',
            top: 4,
            right: 4,
            display: 'none',
          }}
          className="slot-card-actions"
        >
          <Tooltip title="删除">
            <Button
              type="text"
              danger
              size="small"
              icon={<DeleteOutlined />}
              onClick={(e) => {
                e.stopPropagation();
                handleDelete(slot.id);
              }}
            />
          </Tooltip>
        </div>
      </div>
    );
  }

  function renderModal() {
    return (
      <Modal
        title={editingSlot ? '编辑时间段' : '新建时间段'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmit}
        confirmLoading={submitLoading}
        okText={editingSlot ? '保存' : '创建'}
        cancelText="取消"
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label="名称"
            rules={[
              { required: true, message: '请输入时间段名称' },
              { min: 2, max: 50, message: '名称长度 2-50 个字符' },
            ]}
          >
            <Input placeholder="如「上午第1节」" />
          </Form.Item>
          <Form.Item
            name="day_of_week"
            label="星期"
            rules={[{ required: true, message: '请选择星期' }]}
          >
            <Select options={WEEKDAYS} placeholder="请选择星期" />
          </Form.Item>
          <Form.Item
            name="time_range"
            label="时间范围"
            rules={[{ required: true, message: '请选择时间范围' }]}
          >
            <TimePicker.RangePicker format="HH:mm" minuteStep={5} style={{ width: '100%' }} />
          </Form.Item>
          {editingSlot && (
            <Form.Item name="is_active" label="启用状态" valuePropName="checked" initialValue={true}>
              <Select
                options={[
                  { label: '启用', value: true },
                  { label: '停用', value: false },
                ]}
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    );
  }

  return (
    <div>
      <Flex justify="flex-end" style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openCreate()}>
          新建时间段
        </Button>
      </Flex>

      {/* 周视图网格 */}
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
        <div
          style={{
            background: '#fafafa',
            padding: '10px 8px',
            textAlign: 'center',
            fontWeight: 600,
            fontSize: 13,
          }}
        >
          时间
        </div>
        {WEEKDAYS.map((day) => (
          <div
            key={day.value}
            style={{
              background: '#fafafa',
              padding: '10px 8px',
              textAlign: 'center',
              fontWeight: 600,
              fontSize: 13,
            }}
          >
            {day.label}
          </div>
        ))}

        {/* 数据行 */}
        {timeRows.map((timeRange) => {
          const rowKey = `${timeRange.start}-${timeRange.end}`;
          return [
            // 时间列
            <div
              key={`time-${rowKey}`}
              style={{
                background: '#fff',
                padding: '12px 8px',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Text style={{ fontSize: 13, fontWeight: 500 }}>{timeRange.start}</Text>
              <Text type="secondary" style={{ fontSize: 11 }}>~</Text>
              <Text style={{ fontSize: 13, fontWeight: 500 }}>{timeRange.end}</Text>
            </div>,
            // 每天的格子
            ...WEEKDAYS.map((day) => {
              const slot = slotMap.get(`${rowKey}_${day.value}`);
              return (
                <div
                  key={`${rowKey}_${day.value}`}
                  style={{
                    background: '#fff',
                    padding: 6,
                    minHeight: 80,
                    display: 'flex',
                    alignItems: 'stretch',
                    cursor: slot ? 'default' : 'pointer',
                  }}
                  onClick={() => {
                    if (!slot) {
                      form.resetFields();
                      form.setFieldsValue({
                        day_of_week: day.value,
                        time_range: [
                          dayjs(timeRange.start, 'HH:mm'),
                          dayjs(timeRange.end, 'HH:mm'),
                        ],
                      });
                      setEditingSlot(null);
                      setModalOpen(true);
                    }
                  }}
                >
                  {slot ? (
                    <div style={{ flex: 1 }}>{renderSlotCard(slot)}</div>
                  ) : (
                    <Tooltip title="点击添加时间段">
                      <div
                        style={{
                          flex: 1,
                          borderRadius: 8,
                          border: '1px dashed #e8e8e8',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          transition: 'all 0.2s',
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.borderColor = '#1677ff';
                          e.currentTarget.style.background = '#f0f5ff';
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.borderColor = '#e8e8e8';
                          e.currentTarget.style.background = 'transparent';
                        }}
                      >
                        <PlusOutlined style={{ color: '#bbb', fontSize: 16 }} />
                      </div>
                    </Tooltip>
                  )}
                </div>
              );
            }),
          ];
        })}
      </div>

      {/* CSS for hover effects on slot cards */}
      <style>{`
        .slot-card:hover .slot-card-actions {
          display: flex !important;
        }
        .slot-card:hover {
          box-shadow: 0 2px 8px rgba(0,0,0,0.08);
        }
      `}</style>

      {renderModal()}
    </div>
  );
}
