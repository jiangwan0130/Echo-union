import { test, expect } from '@playwright/test';

test.describe('认证流程', () => {
  test('应显示登录页面', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByRole('heading', { level: 4 })).toBeVisible();
    await expect(page.getByPlaceholder(/学号/)).toBeVisible();
    await expect(page.getByPlaceholder(/密码/)).toBeVisible();
  });

  test('未登录访问受保护页面应跳转到登录页', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForURL('**/login');
    await expect(page).toHaveURL(/\/login/);
  });

  test('错误凭证应显示错误提示', async ({ page }) => {
    await page.goto('/login');
    await page.getByPlaceholder(/学号/).fill('wrong_id');
    await page.getByPlaceholder(/密码/).fill('wrong_password');
    await page.getByRole('button', { name: /登录/ }).click();
    // 等待错误提示（antd message）
    await expect(page.locator('.ant-message')).toBeVisible({ timeout: 5000 });
  });

  test('访问 /register 应返回 404', async ({ page }) => {
    await page.goto('/register');
    // 注册路由已移除，应进入 404
    await expect(page.getByText(/404|不存在|找不到/)).toBeVisible({
      timeout: 5000,
    });
  });
});
