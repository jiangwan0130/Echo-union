import { useState } from 'react';
import { Modal, Form, Select, TimePicker, Input } from 'antd';
import dayjs from 'dayjs';
import type { UnavailableTime, CreateUnavailableTimeRequest, UpdateUnavailableTimeRequest } from '@/types';

const WEEKDAYS = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 7, label: '周日' },
];

const REPEAT_TYPES = [
  { value: 'weekly', label: '每周' },
  { value: 'biweekly', label: '隔周' },
  { value: 'once', label: '仅一次' },
];

const WEEK_TYPES = [
  { value: 'all', label: '所有周' },
  { value: 'odd', label: '单周' },
  { value: 'even', label: '双周' },
];

interface UnavailableTimeModalProps {
  open: boolean;
  editingItem: UnavailableTime | null;
  onCancel: () => void;
  onSubmit: (data: CreateUnavailableTimeRequest | UpdateUnavailableTimeRequest) => Promise<void>;
  /** 预填的星期和时间（从网格点击触发时传入） */
  prefill?: { dayOfWeek?: number; startTime?: string; endTime?: string };
}

/**
 * 添加/编辑不可用时间弹窗
 */
export default function UnavailableTimeModal({
  open,
  editingItem,
  onCancel,
  onSubmit,
  prefill,
}: UnavailableTimeModalProps) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const isEdit = !!editingItem;

  const handleOpen = () => {
    if (editingItem) {
      form.setFieldsValue({
        day_of_week: editingItem.day_of_week,
        time_range: [
          dayjs(editingItem.start_time.replace(/:\d{2}$/, ''), 'HH:mm'),
          dayjs(editingItem.end_time.replace(/:\d{2}$/, ''), 'HH:mm'),
        ],
        repeat_type: editingItem.repeat_type || 'weekly',
        week_type: editingItem.week_type || 'all',
        reason: editingItem.reason || '',
      });
    } else {
      form.resetFields();
      if (prefill) {
        const values: Record<string, unknown> = {};
        if (prefill.dayOfWeek) values.day_of_week = prefill.dayOfWeek;
        if (prefill.startTime && prefill.endTime) {
          values.time_range = [
            dayjs(prefill.startTime, 'HH:mm'),
            dayjs(prefill.endTime, 'HH:mm'),
          ];
        }
        form.setFieldsValue(values);
      }
    }
  };

  const handleSubmit = async () => {
    const values = await form.validateFields();
    const startTime = values.time_range[0].format('HH:mm');
    const endTime = values.time_range[1].format('HH:mm');

    setLoading(true);
    try {
      await onSubmit({
        day_of_week: values.day_of_week,
        start_time: startTime,
        end_time: endTime,
        reason: values.reason || undefined,
        repeat_type: values.repeat_type || 'weekly',
        week_type: values.week_type || 'all',
      });
      form.resetFields();
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={isEdit ? '编辑不可用时间' : '添加不可用时间'}
      open={open}
      onCancel={onCancel}
      onOk={handleSubmit}
      confirmLoading={loading}
      okText={isEdit ? '保存' : '添加'}
      cancelText="取消"
      destroyOnClose
      afterOpenChange={(visible) => {
        if (visible) handleOpen();
      }}
    >
      <Form
        form={form}
        layout="vertical"
        style={{ marginTop: 16 }}
        initialValues={{ repeat_type: 'weekly', week_type: 'all' }}
      >
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

        <Form.Item name="repeat_type" label="重复类型">
          <Select options={REPEAT_TYPES} />
        </Form.Item>

        <Form.Item name="week_type" label="周类型">
          <Select options={WEEK_TYPES} />
        </Form.Item>

        <Form.Item name="reason" label="备注">
          <Input.TextArea placeholder="如：社团活动、兼职等（可选）" rows={2} maxLength={200} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
