import { Tabs } from 'antd';
import {
  CalendarOutlined,
  ClockCircleOutlined,
  EnvironmentOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { PageHeader } from '@/components/common';
import SemesterTab from './SemesterTab';
import TimeSlotTab from './TimeSlotTab';
import LocationTab from './LocationTab';
import ScheduleRuleTab from './ScheduleRuleTab';
import SystemParamTab from './SystemParamTab';

export default function ConfigPage() {
  const tabItems = [
    {
      key: 'semester',
      label: (
        <span><CalendarOutlined style={{ marginRight: 6 }} />学期管理</span>
      ),
      children: <SemesterTab />,
    },
    {
      key: 'timeslot',
      label: (
        <span><ClockCircleOutlined style={{ marginRight: 6 }} />时间段配置</span>
      ),
      children: <TimeSlotTab />,
    },
    {
      key: 'location',
      label: (
        <span><EnvironmentOutlined style={{ marginRight: 6 }} />值班地点</span>
      ),
      children: <LocationTab />,
    },
    {
      key: 'rule',
      label: (
        <span><SafetyCertificateOutlined style={{ marginRight: 6 }} />排班规则</span>
      ),
      children: <ScheduleRuleTab />,
    },
    {
      key: 'system',
      label: (
        <span><SettingOutlined style={{ marginRight: 6 }} />系统参数</span>
      ),
      children: <SystemParamTab />,
    },
  ];

  return (
    <div>
      <PageHeader
        title="系统配置"
        description="管理学期、时间段、地点与排班规则等系统级配置"
      />
      <Tabs items={tabItems} defaultActiveKey="semester" />
    </div>
  );
}
