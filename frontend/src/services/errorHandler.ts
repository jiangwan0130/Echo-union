import { message } from 'antd';
import type { AxiosError } from 'axios';
import type { ApiResponse } from '@/types';

/**
 * 后端业务错误码 → 前端友好提示映射
 */
const ERROR_CODE_MAP: Record<number, string> = {
  // ── 通用 ──
  10001: '参数校验失败',
  10003: '无权操作',
  10004: '请求过于频繁，请稍后再试',
  10005: '请求体过大',

  // ── 认证模块 11xxx ──
  11001: '学号或密码错误',
  11002: 'Token已过期',
  11003: 'Token无效或已被吊销',
  11005: '邮箱已被注册',
  11006: '学号已被注册',

  // ── 用户模块 12xxx ──
  12001: '用户不存在',
  12002: '无法修改自己的角色',
  12003: '无法删除自己',
  12004: '邮箱已被使用',
  12005: '部门不存在',
  12006: '学号已被使用',

  // ── 部门模块 14xxx ──
  // 注意：学期模块复用 14001-14003 号段，后端返回的 message 会更具体
  14001: '资源不存在',
  14002: '名称已存在',
  14003: '存在关联数据，无法删除',
  14004: '部门已停用',
  14005: '指定用户不属于该部门',
  14006: '指定用户不存在',
  14007: '学期不存在',

  // ── 时间表模块 15xxx ──
  15001: '时间段不存在',
  15002: '无活动学期',
  15003: '未找到学期分配记录',
  15004: '时间表已提交，不可修改',
  15005: '尚未导入课表或标记不可用时间',
  15006: 'ICS 文件解析失败',
  15007: 'ICS 文件中无有效课程',
  15008: '不可用时间记录不存在',
  15009: '无权操作该记录',
  15010: '部门不存在',

  // ── 地点模块 16xxx ──
  16001: '地点不存在',

  // ── 导出模块 161xx ──
  16101: '该学期暂无排班表',
  16102: '排班表中无排班项',

  // ── 系统配置 17xxx ──
  17001: '系统配置未初始化',

  // ── 排班规则 18xxx ──
  18001: '排班规则不存在',
  18002: '该规则不可配置',

  // ── 排班 13xxx ──
  13101: '排班表不存在',
  13102: '排班项不存在',
  13103: '该学期已存在排班表',
  13104: '排班表非草稿状态，不可执行此操作',
  13105: '排班表非已发布状态',
  13106: '排班表不可发布',
  13107: '课表提交率未达100%，请确保所有成员已提交课表',
  13108: '无符合条件的排班候选人',
  13109: '无可用时间段',
  13110: '候选人在该时段不可用',
  13111: '学期不存在',
};

/**
 * 从 Axios 错误中提取后端业务错误信息
 */
export function extractErrorMessage(error: unknown, fallback = '操作失败'): string {
  const axiosError = error as AxiosError<ApiResponse>;
  const resp = axiosError?.response?.data;

  if (resp?.code && ERROR_CODE_MAP[resp.code]) {
    return ERROR_CODE_MAP[resp.code];
  }

  if (resp?.message && resp.message !== 'error') {
    return resp.message;
  }

  return fallback;
}

/**
 * 提取后端错误码
 */
export function extractErrorCode(error: unknown): number | undefined {
  const axiosError = error as AxiosError<ApiResponse>;
  return axiosError?.response?.data?.code;
}

/**
 * 显示后端错误提示（自动映射错误码）
 */
export function showError(error: unknown, fallback = '操作失败') {
  message.error(extractErrorMessage(error, fallback));
}
