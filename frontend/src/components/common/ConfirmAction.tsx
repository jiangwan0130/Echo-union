import { Popconfirm } from 'antd';
import type { ReactNode } from 'react';

interface ConfirmActionProps {
  title: string;
  description?: string;
  onConfirm: () => void | Promise<void>;
  children: ReactNode;
  danger?: boolean;
}

export default function ConfirmAction({
  title,
  description,
  onConfirm,
  children,
  danger = true,
}: ConfirmActionProps) {
  return (
    <Popconfirm
      title={title}
      description={description}
      onConfirm={onConfirm}
      okText="确认"
      cancelText="取消"
      okButtonProps={{ danger }}
    >
      {children}
    </Popconfirm>
  );
}
