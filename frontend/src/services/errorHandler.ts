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

  // ── 用户模块 12xxx ──
  12001: '用户不存在',
  12002: '无法修改自己的角色',
  12003: '无法删除自己',
  12004: '邮箱已被使用',
  12005: '部门不存在',

  // ── 部门模块 14xxx ──
  14001: '部门不存在',
  14002: '部门名称已存在',
  14003: '部门下存在成员，无法删除',
  14004: '部门已停用',
  14005: '指定用户不属于该部门',
  14006: '指定用户不存在',
  14007: '学期不存在',

  // ── 学期模块 ──
  // 14001 复用
  // 14002: '学期日期无效' — 注意与部门共用编号段，后端实际返回的 message 会更准确
  // 14003: '学期日期与已有学期重叠'

  // ── 时间段模块 15xxx ──
  15001: '时间段不存在',
  15002: '关联的学期不存在',

  // ── 地点模块 16xxx ──
  16001: '地点不存在',

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
