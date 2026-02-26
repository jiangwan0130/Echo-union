import { useState } from 'react';
import { Card, Upload, Input, Button, Space, Typography, message, Flex } from 'antd';
import { UploadOutlined, LinkOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd';
import { timetableApi, showError } from '@/services';

const { Text } = Typography;

interface CourseImportSectionProps {
  semesterId?: string;
  courseCount: number;
  onImported: () => void;
}

/**
 * ICS 课表导入区域
 * 支持两种方式：文件上传(.ics) 和 URL 链接导入
 */
export default function CourseImportSection({
  semesterId,
  courseCount,
  onImported,
}: CourseImportSectionProps) {
  const [icsUrl, setIcsUrl] = useState('');
  const [urlLoading, setUrlLoading] = useState(false);
  const [fileLoading, setFileLoading] = useState(false);

  // 文件上传处理
  const handleFileUpload = async (file: UploadFile) => {
    if (!file.originFileObj) return;
    const formData = new FormData();
    formData.append('file', file.originFileObj);
    if (semesterId) {
      formData.append('semester_id', semesterId);
    }

    setFileLoading(true);
    try {
      const { data } = await timetableApi.importICS(formData);
      message.success(`成功导入 ${data.data.imported_count} 门课程`);
      onImported();
    } catch (err) {
      showError(err, '课表导入失败');
    } finally {
      setFileLoading(false);
    }
  };

  // URL 链接导入
  const handleUrlImport = async () => {
    if (!icsUrl.trim()) {
      message.warning('请输入 ICS 链接');
      return;
    }
    setUrlLoading(true);
    try {
      const { data } = await timetableApi.importICS({
        url: icsUrl.trim(),
        semester_id: semesterId,
      });
      message.success(`成功导入 ${data.data.imported_count} 门课程`);
      setIcsUrl('');
      onImported();
    } catch (err) {
      showError(err, '课表导入失败');
    } finally {
      setUrlLoading(false);
    }
  };

  return (
    <Card
      title="📥 导入课程表"
      size="small"
      extra={
        courseCount > 0 ? (
          <Text type="success">已导入 {courseCount} 门课程</Text>
        ) : null
      }
    >
      <Flex gap={16} wrap="wrap" align="flex-end">
        {/* 方式一：文件上传 */}
        <Upload
          accept=".ics"
          maxCount={1}
          showUploadList={false}
          beforeUpload={() => false}
          onChange={({ file }) => handleFileUpload(file)}
        >
          <Button icon={<UploadOutlined />} loading={fileLoading}>
            上传 ICS 文件
          </Button>
        </Upload>

        <Text type="secondary">或</Text>

        {/* 方式二：URL 链接 */}
        <Space.Compact style={{ flex: 1, minWidth: 300 }}>
          <Input
            prefix={<LinkOutlined />}
            placeholder="输入 ICS 订阅链接"
            value={icsUrl}
            onChange={(e) => setIcsUrl(e.target.value)}
            onPressEnter={handleUrlImport}
          />
          <Button type="primary" loading={urlLoading} onClick={handleUrlImport}>
            导入
          </Button>
        </Space.Compact>
      </Flex>
    </Card>
  );
}
