import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Card,
  Collapse,
  Table,
  Progress,
  Statistic,
  Flex,
  Select,
  Input,
  Tag,
  Spin,
  Button,
  Empty,
} from 'antd';
import {
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { PageHeader } from '@/components/common';
import { timetableApi, showError } from '@/services';
import { useAuthStore } from '@/stores';
import type {
  TimetableProgressResponse,
  DepartmentProgressResponse,
  DepartmentProgressItem,
  DepartmentMemberStatus,
} from '@/types';

/** 部门详情 = 基础统计 + 成员列表 */
interface DepartmentDetail extends DepartmentProgressItem {
  members?: DepartmentMemberStatus[];
  membersLoading?: boolean;
}

export default function ProgressPage() {
  const { user, isAdmin, isLeader } = useAuthStore();

  // ── 全局进度（admin 使用） ──
  const [globalProgress, setGlobalProgress] =
    useState<TimetableProgressResponse | null>(null);

  // ── 部门详情缓存 ──（展开时按需加载）
  const [departmentDetails, setDepartmentDetails] = useState<
    Record<string, DepartmentDetail>
  >({});

  // ── leader 直接加载的部门数据 ──
  const [leaderDepartment, setLeaderDepartment] =
    useState<DepartmentProgressResponse | null>(null);

  const [loading, setLoading] = useState(false);

  // ── 筛选 ──
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [searchText, setSearchText] = useState('');

  // ── 数据加载 ──
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      if (isAdmin()) {
        const { data } = await timetableApi.getProgress();
        setGlobalProgress(data.data);
        // 重置详情缓存
        setDepartmentDetails({});
      } else if (isLeader() && user?.department?.id) {
        const { data } = await timetableApi.getDepartmentProgress(
          user.department.id,
        );
        setLeaderDepartment(data.data);
      }
    } catch (err) {
      showError(err, '加载提交进度失败');
    } finally {
      setLoading(false);
    }
  }, [isAdmin, isLeader, user?.department?.id]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // ── admin: 展开部门时加载成员明细 ──
  const handleCollapseChange = useCallback(
    async (keys: string | string[]) => {
      const activeKeys = Array.isArray(keys) ? keys : [keys];
      for (const deptId of activeKeys) {
        if (departmentDetails[deptId]?.members) continue; // 已加载

        setDepartmentDetails((prev) => ({
          ...prev,
          [deptId]: {
            ...(prev[deptId] ||
              globalProgress?.departments.find(
                (d) => d.department_id === deptId,
              ))!,
            membersLoading: true,
          },
        }));

        try {
          const { data } = await timetableApi.getDepartmentProgress(deptId);
          setDepartmentDetails((prev) => ({
            ...prev,
            [deptId]: {
              ...prev[deptId],
              members: data.data.members,
              membersLoading: false,
            },
          }));
        } catch (err) {
          showError(err, '加载部门成员进度失败');
          setDepartmentDetails((prev) => ({
            ...prev,
            [deptId]: { ...prev[deptId], membersLoading: false },
          }));
        }
      }
    },
    [departmentDetails, globalProgress?.departments],
  );

  // ── 表格列 ──
  const memberColumns: ColumnsType<DepartmentMemberStatus> = [
    {
      title: '姓名',
      dataIndex: 'name',
      width: 120,
    },
    {
      title: '学号',
      dataIndex: 'student_id',
      width: 140,
    },
    {
      title: '状态',
      dataIndex: 'timetable_status',
      width: 100,
      render: (status: string) =>
        status === 'submitted' ? (
          <Tag icon={<CheckCircleOutlined />} color="success">
            已提交
          </Tag>
        ) : (
          <Tag icon={<CloseCircleOutlined />} color="error">
            未提交
          </Tag>
        ),
    },
    {
      title: '提交时间',
      dataIndex: 'submitted_at',
      width: 180,
      render: (val?: string) =>
        val ? dayjs(val).format('YYYY-MM-DD HH:mm') : '—',
    },
  ];

  // ── 成员筛选逻辑 ──
  const filterMembers = useCallback(
    (members?: DepartmentMemberStatus[]) => {
      if (!members) return [];
      return members.filter((m) => {
        const matchStatus =
          statusFilter === 'all' || m.timetable_status === statusFilter;
        const matchSearch =
          !searchText ||
          m.name.includes(searchText) ||
          m.student_id.includes(searchText);
        return matchStatus && matchSearch;
      });
    },
    [statusFilter, searchText],
  );

  // ── admin: 部门列表 ──
  const departments = useMemo(
    () => globalProgress?.departments || [],
    [globalProgress],
  );

  // ── 统计卡片数据 ──
  const stats = useMemo(() => {
    if (isAdmin() && globalProgress) {
      return {
        total: globalProgress.total,
        submitted: globalProgress.submitted,
        notSubmitted: globalProgress.total - globalProgress.submitted,
        progress: globalProgress.progress,
      };
    }
    if (isLeader() && leaderDepartment) {
      return {
        total: leaderDepartment.total,
        submitted: leaderDepartment.submitted,
        notSubmitted: leaderDepartment.total - leaderDepartment.submitted,
        progress: leaderDepartment.progress,
      };
    }
    return null;
  }, [isAdmin, isLeader, globalProgress, leaderDepartment]);

  // ── 渲染：admin 的各部门折叠面板 ──
  const renderAdminContent = () => {
    if (!departments.length) {
      return <Empty description="暂无部门数据" />;
    }

    const items = departments.map((dept) => {
      const detail = departmentDetails[dept.department_id];
      const members = detail?.members;
      const filtered = filterMembers(members);

      return {
        key: dept.department_id,
        label: (
          <Flex align="center" gap={12} style={{ width: '100%' }}>
            <span style={{ fontWeight: 500 }}>{dept.department_name}</span>
            <Progress
              percent={Math.round(dept.progress)}
              size="small"
              style={{ flex: 1, maxWidth: 200 }}
              status={dept.progress >= 100 ? 'success' : 'active'}
            />
            <Tag>
              {dept.submitted}/{dept.total}
            </Tag>
          </Flex>
        ),
        children: detail?.membersLoading ? (
          <div style={{ textAlign: 'center', padding: 24 }}>
            <Spin />
          </div>
        ) : members ? (
          <Table<DepartmentMemberStatus>
            columns={memberColumns}
            dataSource={filtered}
            rowKey="user_id"
            size="small"
            pagination={filtered.length > 10 ? { pageSize: 10 } : false}
          />
        ) : (
          <div style={{ color: '#999', padding: 12 }}>
            加载中...
          </div>
        ),
      };
    });

    return <Collapse items={items} onChange={handleCollapseChange} />;
  };

  // ── 渲染：leader 的本部门表格 ──
  const renderLeaderContent = () => {
    if (!leaderDepartment) {
      return <Empty description="暂无数据" />;
    }
    const filtered = filterMembers(leaderDepartment.members);
    return (
      <Card
        title={leaderDepartment.department_name}
        size="small"
        style={{ marginTop: 16 }}
      >
        <Table<DepartmentMemberStatus>
          columns={memberColumns}
          dataSource={filtered}
          rowKey="user_id"
          size="small"
          pagination={filtered.length > 10 ? { pageSize: 10 } : false}
        />
      </Card>
    );
  };

  return (
    <div>
      <PageHeader
        title="时间表提交进度"
        description={
          isAdmin()
            ? '查看所有部门成员的时间表提交状态'
            : '查看本部门成员的时间表提交状态'
        }
        extra={
          <Button icon={<ReloadOutlined />} onClick={fetchData}>
            刷新
          </Button>
        }
      />

      <Spin spinning={loading}>
        {/* 统计卡片 */}
        {stats && (
          <Flex gap={16} wrap="wrap" style={{ marginBottom: 24 }}>
            <Card size="small" style={{ minWidth: 140 }}>
              <Statistic title="总人数" value={stats.total} />
            </Card>
            <Card size="small" style={{ minWidth: 140 }}>
              <Statistic
                title="已提交"
                value={stats.submitted}
                styles={{ content: { color: '#52c41a' } }}
              />
            </Card>
            <Card size="small" style={{ minWidth: 140 }}>
              <Statistic
                title="未提交"
                value={stats.notSubmitted}
                styles={
                  stats.notSubmitted > 0
                    ? { content: { color: '#ff4d4f' } }
                    : undefined
                }
              />
            </Card>
            <Card size="small" style={{ minWidth: 140 }}>
              <Statistic
                title="提交率"
                value={stats.progress}
                precision={1}
                suffix="%"
                styles={
                  stats.progress >= 100
                    ? { content: { color: '#52c41a' } }
                    : undefined
                }
              />
            </Card>
          </Flex>
        )}

        {/* 筛选栏 */}
        <Flex gap={12} style={{ marginBottom: 16 }} wrap="wrap">
          <Select
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ width: 140 }}
            options={[
              { label: '全部状态', value: 'all' },
              { label: '已提交', value: 'submitted' },
              { label: '未提交', value: 'not_submitted' },
            ]}
          />
          <Input.Search
            placeholder="搜索姓名或学号"
            allowClear
            style={{ width: 220 }}
            onSearch={setSearchText}
            onChange={(e) => {
              if (!e.target.value) setSearchText('');
            }}
          />
        </Flex>

        {/* 主内容区 */}
        {isAdmin() && renderAdminContent()}
        {isLeader() && renderLeaderContent()}
      </Spin>
    </div>
  );
}
