import { describe, it, expect } from 'vitest';
import { extractErrorMessage, extractErrorCode } from '../errorHandler';

// 构造 AxiosError 模拟对象
function makeAxiosError(code: number, msg: string, status = 400) {
  return {
    response: {
      status,
      data: { code, message: msg, data: null },
    },
    isAxiosError: true,
  };
}

describe('errorHandler', () => {
  describe('extractErrorMessage', () => {
    it('应优先返回已知错误码对应的中文提示', () => {
      const error = makeAxiosError(10001, 'validation failed');
      expect(extractErrorMessage(error)).toBe('参数校验失败');
    });

    it('未知错误码但有 message 时应返回 message', () => {
      const error = makeAxiosError(99999, '自定义错误信息');
      expect(extractErrorMessage(error)).toBe('自定义错误信息');
    });

    it('完全无法解析时应返回 fallback', () => {
      expect(extractErrorMessage(new Error('oops'), '操作失败')).toBe('操作失败');
    });

    it('response.data.message 为 "error" 时不应采用', () => {
      const error = makeAxiosError(99999, 'error');
      expect(extractErrorMessage(error, '默认提示')).toBe('默认提示');
    });
  });

  describe('extractErrorCode', () => {
    it('应返回后端错误码', () => {
      const error = makeAxiosError(12001, '用户不存在');
      expect(extractErrorCode(error)).toBe(12001);
    });

    it('无法解析时应返回 undefined', () => {
      expect(extractErrorCode(new Error('oops'))).toBeUndefined();
    });
  });
});
