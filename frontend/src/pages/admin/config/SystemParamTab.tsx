import { useState, useEffect, useCallback } from 'react';
import {
  Form,
  Input,
  InputNumber,
  Button,
  Spin,
  message,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import { systemConfigApi } from '@/services/configApi';
import { showError } from '@/services/errorHandler';
import type { SystemConfigInfo, UpdateSystemConfigRequest } from '@/types';

const { Text } = Typography;

export default function SystemParamTab() {
  const [form] = Form.useForm<UpdateSystemConfigRequest>();
  const [loading, setLoading] = useState(false);
  const [saveLoading, setSaveLoading] = useState(false);
  const [config, setConfig] = useState<SystemConfigInfo | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await systemConfigApi.get();
      setConfig(data.data);
      form.setFieldsValue({
        swap_deadline_hours: data.data.swap_deadline_hours,
        duty_reminder_time: data.data.duty_reminder_time,
        default_location: data.data.default_location,
        sign_in_window_minutes: data.data.sign_in_window_minutes,
        sign_out_window_minutes: data.data.sign_out_window_minutes,
      });
    } catch (err) {
      showError(err, '加载系统参数失败');
    } finally {
      setLoading(false);
    }
  }, [form]);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaveLoading(true);
      await systemConfigApi.update(values);
      message.success('系统参数已保存');
      fetchConfig();
    } catch (err) {
      showError(err, '保存失败');
    } finally {
      setSaveLoading(false);
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        form={form}
        layout="vertical"
        style={{ maxWidth: 520 }}
      >
        <Form.Item
          name="swap_deadline_hours"
          label="换班申请截止时间（小时）"
          tooltip="距排班开始前多少小时关闭换班申请"
          rules={[{ required: true, message: '请输入' }]}
        >
          <InputNumber min={0} max={168} addonAfter="小时" style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="duty_reminder_time"
          label="值班提醒时间"
          tooltip="每天在此时间发送值班提醒"
        >
          <Input placeholder="如 08:00" style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="default_location"
          label="默认值班地点"
        >
          <Input placeholder="默认值班地点名称" />
        </Form.Item>

        <Form.Item
          name="sign_in_window_minutes"
          label="签到窗口时间（分钟）"
          tooltip="排班开始前后多少分钟内允许签到"
          rules={[{ required: true, message: '请输入' }]}
        >
          <InputNumber min={0} max={120} addonAfter="分钟" style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item
          name="sign_out_window_minutes"
          label="签退窗口时间（分钟）"
          tooltip="排班结束前后多少分钟内允许签退"
          rules={[{ required: true, message: '请输入' }]}
        >
          <InputNumber min={0} max={120} addonAfter="分钟" style={{ width: '100%' }} />
        </Form.Item>

        {config?.updated_at && (
          <Text type="secondary" style={{ display: 'block', marginBottom: 16, fontSize: 12 }}>
            上次更新：{dayjs(config.updated_at).format('YYYY-MM-DD HH:mm:ss')}
          </Text>
        )}

        <Form.Item>
          <Button type="primary" loading={saveLoading} onClick={handleSave}>
            保存配置
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
}
