import { test, expect } from '@playwright/test';

/**
 * 提交进度页面 E2E 测试
 *
 * 注意：此测试需要后端运行且有测试数据。
 * 如后端未启动，测试会因登录失败而跳过关键断言。
 * 可通过 API mock 或 seed 数据来解决。
 */
test.describe('提交进度页面', () => {
  // 以下测试需要登录态，可通过 beforeEach 设置
  // 由于 E2E 需要真实后端，这里仅验证页面路由和基本 UI 结构

  test('未登录访问 /admin/progress 应跳转到登录页', async ({ page }) => {
    await page.goto('/admin/progress');
    await page.waitForURL('**/login');
    await expect(page).toHaveURL(/\/login/);
  });
});
