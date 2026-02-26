import { useState } from 'react';
import { Modal, Input, Typography } from 'antd';

const { Text } = Typography;

interface PublishModalProps {
  open: boolean;
  /** 'publish' = 草稿→发布, 'modify' = 已发布→修改确认 */
  mode: 'publish' | 'modify';
  onCancel: () => void;
  onConfirm: (reason?: string) => Promise<void>;
}

/**
 * 发布确认弹窗 / 发布后修改确认弹窗
 */
export default function PublishModal({
  open,
  mode,
  onCancel,
  onConfirm,
}: PublishModalProps) {
  const [reason, setReason] = useState('');
  const [loading, setLoading] = useState(false);

  const handleOk = async () => {
    if (mode === 'modify' && !reason.trim()) return;
    setLoading(true);
    try {
      await onConfirm(mode === 'modify' ? reason.trim() : undefined);
      setReason('');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={mode === 'publish' ? '发布排班表' : '确认修改已发布排班'}
      open={open}
      onCancel={onCancel}
      onOk={handleOk}
      confirmLoading={loading}
      okText={mode === 'publish' ? '确认发布' : '确认修改'}
      okButtonProps={
        mode === 'modify' ? { disabled: !reason.trim() } : undefined
      }
    >
      {mode === 'publish' ? (
        <Text>
          发布后所有成员将可以查看排班表。确认发布？
        </Text>
      ) : (
        <>
          <Text style={{ display: 'block', marginBottom: 12 }}>
            排班表已发布，修改后将生成变更记录。
          </Text>
          <Text strong style={{ display: 'block', marginBottom: 4 }}>
            修改原因 <Text type="danger">*</Text>
          </Text>
          <Input.TextArea
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="请填写修改原因"
            rows={3}
            maxLength={200}
          />
        </>
      )}
    </Modal>
  );
}
