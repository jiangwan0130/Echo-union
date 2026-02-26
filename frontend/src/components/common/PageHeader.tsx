import type { ReactNode } from 'react';
import { Typography, Flex } from 'antd';

const { Title, Text } = Typography;

interface PageHeaderProps {
  title: string;
  description?: string;
  extra?: ReactNode;
}

export default function PageHeader({ title, description, extra }: PageHeaderProps) {
  return (
    <Flex
      justify="space-between"
      align="flex-start"
      wrap="wrap"
      gap={16}
      style={{ marginBottom: 24 }}
    >
      <div>
        <Title level={4} style={{ margin: 0 }}>
          {title}
        </Title>
        {description && (
          <Text type="secondary" style={{ marginTop: 4, display: 'block' }}>
            {description}
          </Text>
        )}
      </div>
      {extra && <Flex gap={8} wrap="wrap">{extra}</Flex>}
    </Flex>
  );
}
