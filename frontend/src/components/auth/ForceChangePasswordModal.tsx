import { useState } from 'react';
import { Modal, Form, Input, message, Typography } from 'antd';
import { LockOutlined } from '@ant-design/icons';
import { authApi, showError } from '@/services';
import { useAuthStore } from '@/stores';

const { Text } = Typography;

/**
 * 强制修改密码弹窗
 * 当 user.must_change_password === true 时由 AuthGuard 渲染，不可关闭。
 * 修改成功后刷新用户信息，must_change_password 变为 false，弹窗自动消失。
 */
export default function ForceChangePasswordModal() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const { fetchCurrentUser } = useAuthStore();

  const handleSubmit = async () => {
    const values = await form.validateFields();
    setLoading(true);
    try {
      await authApi.changePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      });
      message.success('密码修改成功');
      // 刷新用户信息，must_change_password 将变为 false
      await fetchCurrentUser();
    } catch (err) {
      showError(err, '密码修改失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="首次登录 — 请修改密码"
      open
      closable={false}
      maskClosable={false}
      keyboard={false}
      okText="确认修改"
      cancelButtonProps={{ style: { display: 'none' } }}
      onOk={handleSubmit}
      confirmLoading={loading}
    >
      <div style={{ marginBottom: 16 }}>
        <Text type="secondary">
          您的账户使用初始密码，为保障安全请立即修改。
        </Text>
      </div>

      <Form form={form} layout="vertical" autoComplete="off">
        <Form.Item
          name="old_password"
          label="当前密码"
          rules={[{ required: true, message: '请输入当前密码' }]}
        >
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="输入当前密码（初始密码为 Ec+学号后6位）"
          />
        </Form.Item>

        <Form.Item
          name="new_password"
          label="新密码"
          rules={[
            { required: true, message: '请输入新密码' },
            { min: 8, message: '密码至少8个字符' },
            { max: 20, message: '密码最多20个字符' },
            {
              pattern: /^(?=.*[a-zA-Z])(?=.*\d)/,
              message: '密码须包含字母和数字',
            },
          ]}
        >
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="8-20位，须包含字母和数字"
          />
        </Form.Item>

        <Form.Item
          name="confirm_password"
          label="确认新密码"
          dependencies={['new_password']}
          rules={[
            { required: true, message: '请确认新密码' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('new_password') === value) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error('两次输入的密码不一致'));
              },
            }),
          ]}
        >
          <Input.Password
            prefix={<LockOutlined />}
            placeholder="再次输入新密码"
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
