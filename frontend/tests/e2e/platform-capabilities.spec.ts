import { expect, test } from '@playwright/test'
import { desktopCapabilities, installRuntime, iosCapabilities } from './runtime'

test('iOS startup excludes desktop-only features and side effects', async ({ page }) => {
  await installRuntime(page, iosCapabilities)
  await page.goto('/')

  await expect(page.locator('html')).toHaveAttribute('data-runtime-target', 'ios')
  await expect(page.locator('html')).toHaveAttribute('data-form-factor', 'phone')
  await expect(page.getByTestId('plugins-view-container')).toHaveCount(0)
  await expect(page.getByTestId('custom-storage-settings')).toHaveCount(0)
  await expect(page.getByTestId('self-update-settings')).toHaveCount(0)
  await expect.poll(() => page.evaluate(() => (
    window as typeof window & { __HAYA_MIDI_REQUESTS__?: number }
  ).__HAYA_MIDI_REQUESTS__)).toBe(0)
})

test('desktop startup retains its existing platform features', async ({ page }) => {
  await installRuntime(page, desktopCapabilities)
  await page.goto('/')

  await expect(page.locator('html')).toHaveAttribute('data-runtime-target', 'desktop')
  await expect(page.locator('html')).toHaveAttribute('data-form-factor', 'desktop')
  await expect(page.getByTestId('plugins-view-container')).toHaveCount(1)
  await expect(page.getByTestId('custom-storage-settings')).toHaveCount(1)
  await expect(page.getByTestId('self-update-settings')).toHaveCount(1)
  await expect.poll(() => page.evaluate(() => (
    window as typeof window & { __HAYA_MIDI_REQUESTS__?: number }
  ).__HAYA_MIDI_REQUESTS__)).toBe(1)
})
