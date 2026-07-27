import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'
import {
  androidCapabilities,
  desktopCapabilities,
  installBackend,
  installRuntime,
  iosCapabilities,
  iosTabletCapabilities,
} from './runtime'

async function assertSafeAreaShell(page: Page) {
  const shellContract = await page.locator('#app-layout').evaluate((shell) => {
    const rootStyle = getComputedStyle(document.documentElement)
    const shellStyle = getComputedStyle(shell)
    return {
      safeTop: rootStyle.getPropertyValue('--safe-top').trim(),
      safeRight: rootStyle.getPropertyValue('--safe-right').trim(),
      safeBottom: rootStyle.getPropertyValue('--safe-bottom').trim(),
      safeLeft: rootStyle.getPropertyValue('--safe-left').trim(),
      minimumTouchTarget: rootStyle.getPropertyValue('--minimum-touch-target').trim(),
      paddingTop: shellStyle.paddingTop,
      paddingRight: shellStyle.paddingRight,
      paddingBottom: shellStyle.paddingBottom,
      paddingLeft: shellStyle.paddingLeft,
    }
  })

  expect(shellContract.safeTop).not.toBe('')
  expect(shellContract.safeRight).not.toBe('')
  expect(shellContract.safeBottom).not.toBe('')
  expect(shellContract.safeLeft).not.toBe('')
  expect(shellContract.minimumTouchTarget).toBe('44px')
  expect(shellContract.paddingTop).toBe('0px')
  expect(shellContract.paddingRight).toBe('0px')
  expect(shellContract.paddingBottom).toBe('0px')
  expect(shellContract.paddingLeft).toBe('0px')

  const firstVisibleButton = page.locator('button:visible').first()
  const buttonBox = await firstVisibleButton.boundingBox()
  expect(buttonBox).not.toBeNull()
  expect(buttonBox!.height).toBeGreaterThanOrEqual(44)
  expect(buttonBox!.width).toBeGreaterThanOrEqual(44)
}

test('iOS startup excludes desktop-only features and side effects', async ({ page }) => {
  await installRuntime(page, iosCapabilities)
  await page.goto('/')

  await expect(page.locator('html')).toHaveAttribute('data-runtime-target', 'ios')
  await expect(page.locator('html')).toHaveAttribute('data-form-factor', 'phone')
  await expect(page.getByTestId('plugins-view-container')).toHaveCount(0)
  await expect(page.getByTestId('custom-storage-settings')).toHaveCount(0)
  await expect(page.getByTestId('self-update-settings')).toHaveCount(0)
  await expect(page.getByRole('navigation', { name: 'Primary' })).toHaveCount(0)
  await expect.poll(() => page.evaluate(() => (
    window as typeof window & { __HAYA_MIDI_REQUESTS__?: number }
  ).__HAYA_MIDI_REQUESTS__)).toBe(0)

  await page.evaluate(() => {
    window.dispatchEvent(new CustomEvent('nativeTabSelected', { detail: { index: 3 } }))
  })
  await expect(page.locator('#view-settings')).not.toHaveClass(/hidden/)
})

test('Android renders localized web top-level navigation', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await installRuntime(page, androidCapabilities)
  await page.goto('/')

  const navigation = page.getByRole('navigation', { name: 'Primary' })
  await expect(navigation).toBeVisible()
  await expect(navigation.getByRole('button', { name: 'Library' })).toBeVisible()
  await expect(navigation.getByRole('button', { name: 'Offline' })).toBeVisible()
  await expect(navigation.getByRole('button', { name: 'Search' })).toBeVisible()
  await expect(navigation.getByRole('button', { name: 'Settings' })).toBeVisible()
  const navigationBox = await navigation.boundingBox()
  const mainContentBox = await page.locator('#main-content').boundingBox()
  expect(navigationBox).not.toBeNull()
  expect(mainContentBox).not.toBeNull()
  expect(navigationBox!.y).toBeGreaterThanOrEqual(mainContentBox!.y + mainContentBox!.height)

  await page.locator('#app').evaluate((app) => {
    const pinia = (app as HTMLElement & {
      __vue_app__: { config: { globalProperties: { $pinia: any } } }
    }).__vue_app__.config.globalProperties.$pinia
    pinia._s.get('tabs').currentCategoryId = 'category-1'
  })
  await navigation.getByRole('button', { name: 'Library' }).click()
  await expect.poll(() => page.locator('#app').evaluate((app) => {
    const pinia = (app as HTMLElement & {
      __vue_app__: { config: { globalProperties: { $pinia: any } } }
    }).__vue_app__.config.globalProperties.$pinia
    return pinia._s.get('tabs').currentCategoryId
  })).toBe('')

  await page.locator('#app').evaluate((app) => {
    const pinia = (app as HTMLElement & {
      __vue_app__: { config: { globalProperties: { $pinia: any } } }
    }).__vue_app__.config.globalProperties.$pinia
    const tabs = pinia._s.get('tabs')
    const makeTab = (id: string, title: string, isCloud: boolean) => ({
      id,
      title,
      artist: '',
      album: '',
      filePath: `${id}.pdf`,
      type: 'pdf',
      isManaged: !isCloud,
      isCloud,
      coverPath: '',
      categoryIds: [],
      country: '',
      language: '',
      originCountry: '',
      tag: '',
      addedAt: 0,
      lastOpened: 0,
      initialAz: title[0],
      initialKana: title[0],
    })
    tabs.addTabsInPlace([
      makeTab('local-tab', 'Local Tab', false),
      makeTab('cloud-tab', 'Cloud Tab', true),
    ])
  })

  await navigation.getByRole('button', { name: 'Offline' }).click()
  await expect(page.getByText('Local Tab', { exact: true })).toBeVisible()
  await expect(page.getByText('Cloud Tab', { exact: true })).not.toBeVisible()

  await navigation.getByRole('button', { name: 'Search' }).click()
  await expect(page.getByRole('textbox', { name: 'Search...' })).toBeFocused()

  await navigation.getByRole('button', { name: 'Settings' }).click()
  await expect(page.locator('#view-settings')).not.toHaveClass(/hidden/)
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

test('iOS viewer requests an in-process content URL', async ({ page }) => {
  await installRuntime(page, iosCapabilities)
  await installBackend(page, { GetTabContentURL: '/api/file/tab-1' })
  await page.goto('/')

  await page.locator('#app').evaluate((app) => {
    const pinia = (app as HTMLElement & {
      __vue_app__: { config: { globalProperties: { $pinia: any } } }
    }).__vue_app__.config.globalProperties.$pinia
    const tabs = pinia._s.get('tabs')
    const viewers = pinia._s.get('viewers')
    const ui = pinia._s.get('ui')
    const tab = {
      id: 'tab-1',
      title: 'Mobile PDF',
      artist: '',
      album: '',
      filePath: 'mobile.pdf',
      type: 'pdf',
      isManaged: true,
      isCloud: false,
      coverPath: '',
      categoryIds: [],
      country: '',
      language: '',
      originCountry: '',
      tag: '',
      addedAt: 0,
      lastOpened: 0,
      initialAz: 'M',
      initialKana: 'M',
    }

    tabs.addTabsInPlace([tab])
    viewers.openTab(tab)
    ui.switchView('pdf-tab-1')
  })

  const iframe = page.locator('#pdf-view-tab-1 iframe')
  await expect(iframe).toHaveAttribute('src', /file=%2Fapi%2Ffile%2Ftab-1/)
  await expect.poll(() => page.evaluate(() => (
    window as typeof window & {
      __HAYA_BACKEND_CALLS__?: Array<{ method: string; args: unknown[] }>
    }
  ).__HAYA_BACKEND_CALLS__)).toContainEqual({ method: 'GetTabContentURL', args: ['tab-1'] })
  expect(await page.locator('body').innerHTML()).not.toContain('127.0.0.1')
})

test.describe('adaptive application shell', () => {
  test('iPhone uses a one-column safe-area shell without desktop sidebar or overflow', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await installRuntime(page, iosCapabilities)
    await page.goto('/')

    await expect(page.locator('#sidebar')).toHaveCount(0)
    await assertSafeAreaShell(page)

    await page.locator('#app').evaluate((app) => {
      const pinia = (app as HTMLElement & {
        __vue_app__: { config: { globalProperties: { $pinia: any } } }
      }).__vue_app__.config.globalProperties.$pinia
      const tabs = pinia._s.get('tabs')
      const makeTab = (id: string, title: string) => ({
        id,
        title,
        artist: '',
        album: '',
        filePath: `${id}.pdf`,
        type: 'pdf',
        isManaged: true,
        isCloud: false,
        coverPath: '',
        categoryIds: [],
        country: '',
        language: '',
        originCountry: '',
        tag: '',
        addedAt: 0,
        lastOpened: 0,
        initialAz: 'A',
        initialKana: 'A',
      })
      tabs.addTabsInPlace([makeTab('phone-a', 'Alpha'), makeTab('phone-b', 'Allegro')])
    })

    const cards = page.locator('.tab-card')
    await expect(cards).toHaveCount(2)
    const firstCard = await cards.nth(0).boundingBox()
    const secondCard = await cards.nth(1).boundingBox()
    expect(firstCard).not.toBeNull()
    expect(secondCard).not.toBeNull()
    expect(Math.abs(firstCard!.x - secondCard!.x)).toBeLessThan(1)
    expect(secondCard!.y).toBeGreaterThan(firstCard!.y)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  })

  test('iPad portrait renders a bounded library/sidebar split view', async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 1366 })
    await installRuntime(page, iosTabletCapabilities)
    await page.goto('/')

    const sidebar = page.locator('#sidebar')
    const content = page.locator('#main-content')
    await expect(sidebar).toBeVisible()
    await expect(content).toBeVisible()
    await assertSafeAreaShell(page)

    const sidebarBox = await sidebar.boundingBox()
    const contentBox = await content.boundingBox()
    expect(sidebarBox).not.toBeNull()
    expect(contentBox).not.toBeNull()
    expect(contentBox!.x).toBeGreaterThanOrEqual(sidebarBox!.x + sidebarBox!.width)
    expect(contentBox!.x + contentBox!.width).toBeLessThanOrEqual(1024)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  })

  test('desktop keeps the existing 250px expanded sidebar', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 })
    await installRuntime(page, desktopCapabilities)
    await page.goto('/')

    const sidebar = page.locator('#sidebar')
    await expect(sidebar).toBeVisible()
    await page.locator('#sidebar-toggle').click()
    await expect.poll(() => sidebar.evaluate((element) => getComputedStyle(element).width)).toBe('250px')
  })
})
