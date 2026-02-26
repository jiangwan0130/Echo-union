import { test } from '@playwright/test';

/**
 * 排班相关 E2E 测试
 *
 * 注意：测试需要后端运行且有数据。
 */
test.describe('排班页面', () => {
  test('未登录访问排班页应跳转登录', async ({ page }) => {
    await page.goto('/schedule');
    await page.waitForURL('**/login');
  });

  test('未登录访问自动排班页应跳转登录', async ({ page }) => {
    await page.goto('/schedule/auto');
    await page.waitForURL('**/login');
  });

  test('未登录访问手动调整页应跳转登录', async ({ page }) => {
    await page.goto('/schedule/adjust');
    await page.waitForURL('**/login');
  });
});
