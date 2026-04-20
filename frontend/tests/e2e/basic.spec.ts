import { test, expect } from '@playwright/test';

test.describe('HAYA-TAB Basic App Flow', () => {
  test('has title', async ({ page }) => {
    await page.goto('/');

    // Expect a title "to contain" a substring.
    await expect(page).toHaveTitle(/Vite App|HAYA-TAB/i);
    
    // We should wait for the application to mount, e.g. a root container `#app`
    await expect(page.locator('#app')).toBeVisible();
  });
});
