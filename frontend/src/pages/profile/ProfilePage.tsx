import { useState } from 'react';
import {
  Card,
  Descriptions,
  Form,
  Input,
  Button,
  message,
  Tag,
} from 'antd';
import { LockOutlined, MailOutlined } from '@ant-design/icons';
import { PageHeader } from '@/components/common';
import { authApi, userApi, showError } from '@/services';
import { useAuthStore } from '@/stores';

const ROLE_LABELS: Record<string, string> = {
  admin: '管理员',
  leader: '负责人',
  member: '成员',
};

const ROLE_COLORS: Record<string, string> = {
  admin: 'red',
  leader: 'blue',
  member: 'green',
};

export default function ProfilePage() {
  const { user, fetchCurrentUser } = useAuthStore();
  const [emailForm] = Form.useForm();
  const [pwdForm] = Form.useForm();
  const [emailLoading, setEmailLoading] = useState(false);
  const [pwdLoading, setPwdLoading] = useState(false);

  // 修改邮箱
  const handleUpdateEmail = async (values: { email: string }) => {
    if (!user) return;
    setEmailLoading(true);
    try {
      await userApi.updateUser(user.id, { email: values.email });
      message.success('邮箱已更新');
      await fetchCurrentUser();
    } catch (err) {
      showError(err, '更新失败');
    } finally {
      setEmailLoading(false);
    }
  };

  // 修改密码
  const handleChangePassword = async (values: {
    old_password: string;
    new_password: string;
  }) => {
    setPwdLoading(true);
    try {
      await authApi.changePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      });
      message.success('密码已修改');
      pwdForm.resetFields();
      await fetchCurrentUser();
    } catch (err) {
      showError(err, '修改密码失败');
    } finally {
      setPwdLoading(false);
    }
  };

  if (!user) return null;

  return (
    <div>
      <PageHeader title="个人中心" />

      {/* 基本信息 */}
      <Card title="基本信息" style={{ marginBottom: 16 }}>
        <Descriptions column={{ xs: 1, sm: 2 }} bordered size="small">
          <Descriptions.Item label="姓名">{user.name}</Descriptions.Item>
          <Descriptions.Item label="学号">{user.student_id}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{user.email || '—'}</Descriptions.Item>
          <Descriptions.Item label="角色">
            <Tag color={ROLE_COLORS[user.role] ?? 'default'}>
              {ROLE_LABELS[user.role] ?? user.role}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="所属部门">
            {user.department?.name ?? '未分配'}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* 修改邮箱 */}
      <Card title="修改邮箱" style={{ marginBottom: 16 }}>
        <Form
          form={emailForm}
          layout="inline"
          initialValues={{ email: user.email ?? '' }}
          onFinish={handleUpdateEmail}
        >
          <Form.Item
            name="email"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '请输入有效的邮箱地址' },
            ]}
          >
            <Input
              prefix={<MailOutlined />}
              placeholder="输入新邮箱"
              style={{ width: 280 }}
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={emailLoading}>
              保存
            </Button>
          </Form.Item>
        </Form>
      </Card>

      {/* 修改密码 */}
      <Card title="修改密码">
        <Form
          form={pwdForm}
          layout="vertical"
          onFinish={handleChangePassword}
          style={{ maxWidth: 400 }}
        >
          <Form.Item
            name="old_password"
            label="当前密码"
            rules={[{ required: true, message: '请输入当前密码' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="当前密码" />
          </Form.Item>
          <Form.Item
            name="new_password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 8, max: 20, message: '密码长度 8-20 位' },
              {
                pattern: /^(?=.*[a-zA-Z])(?=.*\d)/,
                message: '密码需包含字母和数字',
              },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="新密码（8-20位，含字母和数字）" />
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
                  return Promise.reject(new Error('两次密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="再次输入新密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={pwdLoading}>
              修改密码
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
