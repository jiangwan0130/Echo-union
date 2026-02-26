import { Tag } from 'antd';

interface StatusTagProps {
  active: boolean;
  activeText?: string;
  inactiveText?: string;
}

export default function StatusTag({
  active,
  activeText = '启用',
  inactiveText = '停用',
}: StatusTagProps) {
  return (
    <Tag color={active ? 'green' : 'default'} bordered={false}>
      {active ? activeText : inactiveText}
    </Tag>
  );
}
