import { useState, useCallback, useEffect } from 'react';
import { Button, Card, Space, Flex, Typography, message, Spin } from 'antd';
import { CheckOutlined, PlusOutlined } from '@ant-design/icons';
import { PageHeader, ConfirmAction } from '@/components/common';
import {
  CourseImportSection,
  UnavailableTimeGrid,
  UnavailableTimeTable,
  UnavailableTimeModal,
} from '@/components/timetable';
import { timetableApi, showError } from '@/services';
import { useAppStore } from '@/stores';
import type {
  MyTimetableResponse,
  UnavailableTime,
  CreateUnavailableTimeRequest,
  UpdateUnavailableTimeRequest,
} from '@/types';

const { Text } = Typography;

export default function TimetablePage() {
  const { currentSemester } = useAppStore();
  const semesterId = currentSemester?.id;

  // ── 数据状态 ──
  const [loading, setLoading] = useState(false);
  const [timetable, setTimetable] = useState<MyTimetableResponse | null>(null);

  // ── Modal 状态 ──
  const [modalOpen, setModalOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<UnavailableTime | null>(null);
  const [prefill, setPrefill] = useState<{
    dayOfWeek?: number;
    startTime?: string;
    endTime?: string;
  }>({});

  // ── 提交状态 ──
  const [submitLoading, setSubmitLoading] = useState(false);

  // ── 快捷计算 ──
  const courses = timetable?.courses || [];
  const unavailable = timetable?.unavailable || [];
  const hasCourses = courses.length > 0;
  const isSubmitted = timetable?.submit_status === 'submitted';

  // ── 加载时间表数据 ──
  const fetchTimetable = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await timetableApi.getMyTimetable(semesterId);
      setTimetable(data.data);
    } catch (err) {
      showError(err, '加载时间表失败');
    } finally {
      setLoading(false);
    }
  }, [semesterId]);

  useEffect(() => {
    fetchTimetable();
  }, [fetchTimetable]);

  // ── 添加/编辑不可用时间 ──
  const handleAddUnavailable = async (
    formData: CreateUnavailableTimeRequest | UpdateUnavailableTimeRequest,
  ) => {
    try {
      if (editingItem) {
        await timetableApi.updateUnavailableTime(
          editingItem.id,
          formData as UpdateUnavailableTimeRequest,
        );
        message.success('不可用时间已更新');
      } else {
        await timetableApi.createUnavailableTime({
          ...formData,
          semester_id: semesterId,
        } as CreateUnavailableTimeRequest);
        message.success('不可用时间已添加');
      }
      setModalOpen(false);
      setEditingItem(null);
      fetchTimetable();
    } catch (err) {
      showError(err, '操作失败');
      throw err;
    }
  };

  // ── 删除不可用时间 ──
  const handleDeleteUnavailable = async (id: string) => {
    try {
      await timetableApi.deleteUnavailableTime(id);
      message.success('已删除');
      fetchTimetable();
    } catch (err) {
      showError(err, '删除失败');
    }
  };

  // ── 提交时间表 ──
  const handleSubmit = async () => {
    setSubmitLoading(true);
    try {
      await timetableApi.submitTimetable({ semester_id: semesterId });
      message.success('时间表已提交');
      fetchTimetable();
    } catch (err) {
      showError(err, '提交失败');
    } finally {
      setSubmitLoading(false);
    }
  };

  // ── 点击空白格 → 打开添加弹窗 ──
  const handleCellClick = (dayOfWeek: number, hour: number) => {
    setEditingItem(null);
    setPrefill({
      dayOfWeek,
      startTime: `${String(hour).padStart(2, '0')}:00`,
      endTime: `${String(hour + 1).padStart(2, '0')}:00`,
    });
    setModalOpen(true);
  };

  // ── 点击不可用色块 → 打开编辑弹窗 ──
  const handleUnavailableClick = (item: UnavailableTime) => {
    setEditingItem(item);
    setPrefill({});
    setModalOpen(true);
  };

  // ── 编辑按钮（列表中） ──
  const handleEdit = (item: UnavailableTime) => {
    setEditingItem(item);
    setPrefill({});
    setModalOpen(true);
  };

  return (
    <div>
      <PageHeader
        title="我的时间表"
        description={
          currentSemester
            ? `当前学期：${currentSemester.name}`
            : '未设置当前学期'
        }
        extra={
          <Space>
            {isSubmitted ? (
              <Text type="success">
                <CheckOutlined /> 已提交
                {timetable?.submitted_at
                  ? ` · ${new Date(timetable.submitted_at).toLocaleDateString()}`
                  : ''}
              </Text>
            ) : (
              <ConfirmAction
                title="确认提交时间表？"
                description="提交后仍可修改不可用时间，但需要重新提交。"
                onConfirm={handleSubmit}
              >
                <Button
                  type="primary"
                  icon={<CheckOutlined />}
                  loading={submitLoading}
                  disabled={!hasCourses}
                >
                  提交时间表
                </Button>
              </ConfirmAction>
            )}
          </Space>
        }
      />

      <Spin spinning={loading}>
        {/* 1. ICS 导入区 */}
        <CourseImportSection
          semesterId={semesterId}
          courseCount={courses.length}
          onImported={fetchTimetable}
        />

        {/* 2. 周视图 */}
        <Card
          title="周视图"
          size="small"
          style={{ marginTop: 16 }}
          extra={
            hasCourses ? (
              <Flex gap={8} align="center">
                <div
                  style={{
                    width: 12,
                    height: 12,
                    background: '#bae0ff',
                    borderRadius: 2,
                  }}
                />
                <Text type="secondary" style={{ fontSize: 12 }}>
                  课程
                </Text>
                <div
                  style={{
                    width: 12,
                    height: 12,
                    background: '#ff4d4f',
                    borderRadius: 2,
                    marginLeft: 8,
                  }}
                />
                <Text type="secondary" style={{ fontSize: 12 }}>
                  不可用
                </Text>
                <div
                  style={{
                    width: 12,
                    height: 12,
                    background: '#fff',
                    border: '1px dashed #d9d9d9',
                    borderRadius: 2,
                    marginLeft: 8,
                  }}
                />
                <Text type="secondary" style={{ fontSize: 12 }}>
                  可用（点击标记）
                </Text>
              </Flex>
            ) : null
          }
        >
          <UnavailableTimeGrid
            courses={courses}
            unavailableTimes={unavailable}
            disabled={!hasCourses}
            onCellClick={handleCellClick}
            onUnavailableClick={handleUnavailableClick}
          />
        </Card>

        {/* 3. 不可用时间列表 */}
        <Card
          title="不可用时间列表"
          size="small"
          style={{ marginTop: 16 }}
          extra={
            <Button
              type="primary"
              size="small"
              icon={<PlusOutlined />}
              disabled={!hasCourses}
              onClick={() => {
                setEditingItem(null);
                setPrefill({});
                setModalOpen(true);
              }}
            >
              添加
            </Button>
          }
        >
          <UnavailableTimeTable
            items={unavailable}
            onEdit={handleEdit}
            onDelete={handleDeleteUnavailable}
          />
        </Card>

        {/* 4. 底部状态栏 */}
        <Flex
          justify="space-between"
          align="center"
          style={{
            marginTop: 16,
            padding: '12px 16px',
            background: '#fafafa',
            borderRadius: 8,
          }}
        >
          <Space size="large">
            <Text>
              课程：<Text strong>{courses.length}</Text> 门
            </Text>
            <Text>
              不可用时段：<Text strong>{unavailable.length}</Text> 个
            </Text>
            <Text>
              状态：
              {isSubmitted ? (
                <Text type="success" strong>
                  已提交
                </Text>
              ) : (
                <Text type="warning" strong>
                  未提交
                </Text>
              )}
            </Text>
          </Space>
        </Flex>
      </Spin>

      {/* 弹窗 */}
      <UnavailableTimeModal
        open={modalOpen}
        editingItem={editingItem}
        onCancel={() => {
          setModalOpen(false);
          setEditingItem(null);
        }}
        onSubmit={handleAddUnavailable}
        prefill={prefill}
      />
    </div>
  );
}
