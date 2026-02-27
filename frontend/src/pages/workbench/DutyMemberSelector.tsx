import { useState, useEffect, useCallback } from 'react';
import {
  Table,
  Checkbox,
  Button,
  Space,
  Tag,
  message,
  Input,
  Statistic,
  Row,
  Col,
  Card,
} from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import { useAppStore } from '@/stores';
import { semesterApi, userApi } from '@/services';
import type { DutyMemberItem } from '@/types';

export default function DutyMemberSelector() {
  const { currentSemester } = useAppStore();
  const semesterId = currentSemester?.id;

  const [members, setMembers] = useState<DutyMemberItem[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [search, setSearch] = useState('');

  const loadMembers = useCallback(async () => {
    if (!semesterId) return;
    setLoading(true);
    try {
      const { data } = await semesterApi.getDutyMembers(semesterId);
      const list = data.data?.list ?? [];
      setMembers(list);
      // 初始化已勾选
      const selected = new Set(
        list.filter((m) => m.duty_required).map((m) => m.user_id),
      );
      setSelectedIds(selected);
    } catch {
      message.error('加载人员列表失败');
    } finally {
      setLoading(false);
    }
  }, [semesterId]);

  useEffect(() => {
    loadMembers();
  }, [loadMembers]);

  const handleToggle = (userId: string, checked: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (checked) {
        next.add(userId);
      } else {
        next.delete(userId);
      }
      return next;
    });
  };

  const handleToggleDepartment = (deptId: string, checked: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      filtered
        .filter((m) => m.department_id === deptId)
        .forEach((m) => {
          if (checked) next.add(m.user_id);
          else next.delete(m.user_id);
        });
      return next;
    });
  };

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedIds(new Set(filtered.map((m) => m.user_id)));
    } else {
      setSelectedIds(new Set());
    }
  };

  const handleSave = async () => {
    if (!semesterId) return;
    setSaving(true);
    try {
      await semesterApi.setDutyMembers(semesterId, {
        user_ids: Array.from(selectedIds),
      });
      message.success(`已保存，共选定 ${selectedIds.size} 名值班人员`);
      await loadMembers();
    } catch {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  // 搜索过滤
  const filtered = members.filter(
    (m) =>
      !search ||
      m.name.includes(search) ||
      m.student_id.includes(search) ||
      m.department_name.includes(search),
  );

  // 按部门分组统计
  const departments = Array.from(
    new Set(filtered.map((m) => m.department_id)),
  ).map((deptId) => {
    const deptMembers = filtered.filter((m) => m.department_id === deptId);
    return {
      id: deptId,
      name: deptMembers[0]?.department_name ?? '未知部门',
      total: deptMembers.length,
      selected: deptMembers.filter((m) => selectedIds.has(m.user_id)).length,
    };
  });

  const columns = [
    {
      title: (
        <Checkbox
          checked={filtered.length > 0 && filtered.every((m) => selectedIds.has(m.user_id))}
          indeterminate={
            filtered.some((m) => selectedIds.has(m.user_id)) &&
            !filtered.every((m) => selectedIds.has(m.user_id))
          }
          onChange={(e) => handleSelectAll(e.target.checked)}
        />
      ),
      dataIndex: 'user_id',
      width: 50,
      render: (userId: string) => (
        <Checkbox
          checked={selectedIds.has(userId)}
          onChange={(e) => handleToggle(userId, e.target.checked)}
        />
      ),
    },
    {
      title: '姓名',
      dataIndex: 'name',
    },
    {
      title: '学号',
      dataIndex: 'student_id',
    },
    {
      title: '部门',
      dataIndex: 'department_name',
      render: (name: string, record: DutyMemberItem) => {
        const dept = departments.find((d) => d.id === record.department_id);
        return (
          <Space>
            <Tag>{name}</Tag>
            {dept && (
              <Checkbox
                checked={dept.selected === dept.total}
                indeterminate={dept.selected > 0 && dept.selected < dept.total}
                onChange={(e) => handleToggleDepartment(record.department_id, e.target.checked)}
              >
                全选部门
              </Checkbox>
            )}
          </Space>
        );
      },
    },
  ];

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="总人数"
              value={members.length}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="已选定"
              value={selectedIds.size}
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title="未选定"
              value={members.length - selectedIds.size}
              valueStyle={{ color: '#999' }}
            />
          </Card>
        </Col>
      </Row>

      <Space style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Input
          placeholder="搜索姓名/学号/部门"
          prefix={<SearchOutlined />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{ width: 300 }}
          allowClear
        />
        <Button type="primary" onClick={handleSave} loading={saving}>
          保存选定人员
        </Button>
      </Space>

      <Table
        dataSource={filtered}
        columns={columns}
        rowKey="user_id"
        loading={loading}
        pagination={{ pageSize: 20 }}
        size="small"
      />
    </div>
  );
}
