import { expect, test, type Page } from '@playwright/test'
import { readFileSync } from 'node:fs'

type DialogName =
  | 'edit'
  | 'category'
  | 'move'
  | 'batchMove'
  | 'confirm'
  | 'keyBindings'
  | 'cloudPicker'
  | 'cloudUpload'
  | 'webdav'
  | 'plugin'

const dialogCases = [
  ['edit', 'medium'],
  ['category', 'small'],
  ['move', 'small'],
  ['batchMove', 'small'],
  ['confirm', 'small'],
  ['keyBindings', 'medium'],
  ['cloudPicker', 'large'],
  ['cloudUpload', 'medium'],
  ['webdav', 'medium'],
  ['plugin', 'small'],
] as const satisfies ReadonlyArray<readonly [DialogName, 'small' | 'medium' | 'large']>

const desktopModalWidths = {
  small: 420,
  medium: 500,
  large: 800,
} as const

async function openDialog(page: Page, dialogName: DialogName) {
  await page.locator('#app').evaluate((app, name) => {
    const pinia = (app as HTMLElement & {
      __vue_app__: { config: { globalProperties: { $pinia: any } } }
    }).__vue_app__.config.globalProperties.$pinia
    const ui = pinia._s.get('ui')
    const settings = pinia._s.get('settings')

    ui.editModalVisible = false
    ui.categoryModalVisible = false
    ui.moveModalVisible = false
    ui.batchMoveModalVisible = false
    ui.confirmModalVisible = false
    ui.keyBindingsModalVisible = false
    ui.cloudPickerModalVisible = false
    ui.cloudUploadModalVisible = false
    ui.webdavModalVisible = false
    ui.pluginModalVisible = false

    settings.settings.webdavEnabled = true
    settings.settings.webdavUrl = ''
    settings.settings.webdavUser = ''
    settings.settings.webdavPassword = ''

    switch (name) {
      case 'edit':
        ui.showEditModal({ title: 'Modal sizing test', type: 'pdf' })
        break
      case 'category':
        ui.showCategoryModal()
        break
      case 'move':
        ui.showMoveModal('test-tab')
        break
      case 'batchMove':
        ui.showBatchMoveModal()
        break
      case 'confirm':
        ui.showConfirmModal('Confirm action', 'This action needs confirmation.', 'Confirm', false, () => {})
        break
      case 'keyBindings':
        ui.showKeyBindingsModal()
        break
      case 'cloudPicker':
        ui.showCloudPickerModal()
        break
      case 'cloudUpload':
        ui.showCloudUploadModal(['C:\\test.pdf'])
        break
      case 'webdav':
        ui.showWebdavModal()
        break
      case 'plugin':
        ui.showPluginModal({
          id: 'test-plugin',
          name: 'Test plugin',
          enabled: true,
          config: { token: '' },
          settingsSchema: { token: 'password' },
        })
        break
    }
  }, dialogName)
}

test.beforeEach(async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('#app')).toBeVisible()
})

test('all real dialogs use the shared semantic size contract', async ({ page }) => {
  for (const [dialogName, expectedSize] of dialogCases) {
    await openDialog(page, dialogName)

    const dialog = page.getByTestId('base-modal')
    await expect(dialog, `${dialogName} should render through BaseModal`).toBeVisible()
    await expect(dialog).toHaveAttribute('data-modal-size', expectedSize)
    await expect(dialog).toHaveAttribute('role', 'dialog')
    await expect(dialog).toHaveAttribute('aria-modal', 'true')
    await expect(dialog).toHaveAttribute('aria-labelledby', /.+/)
    const box = await dialog.boundingBox()
    expect(box).not.toBeNull()
    expect(Math.round(box!.width), `${dialogName} desktop width`).toBe(desktopModalWidths[expectedSize])
  }
})

test('every dialog stays inside a 375 by 667 viewport with sixteen-pixel margins', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 667 })

  for (const [dialogName] of dialogCases) {
    await openDialog(page, dialogName)

    const dialog = page.getByTestId('base-modal')
    await expect(dialog).toBeVisible()
    const box = await dialog.boundingBox()
    expect(box, `${dialogName} should have a measurable dialog box`).not.toBeNull()
    expect(box!.x, `${dialogName} left margin`).toBeGreaterThanOrEqual(15)
    expect(box!.y, `${dialogName} top margin`).toBeGreaterThanOrEqual(15)
    expect(box!.x + box!.width, `${dialogName} right edge`).toBeLessThanOrEqual(360)
    expect(box!.y + box!.height, `${dialogName} bottom edge`).toBeLessThanOrEqual(652)
  }
})

test('Escape closes a dialog and restores focus to its trigger', async ({ page }) => {
  await page.evaluate(() => {
    const trigger = document.createElement('button')
    trigger.id = 'modal-focus-trigger'
    trigger.textContent = 'Open dialog'
    document.body.append(trigger)
    trigger.focus()
  })

  await openDialog(page, 'category')
  const dialog = page.getByTestId('base-modal')
  await expect(dialog).toBeFocused()

  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
  await expect(page.locator('#modal-focus-trigger')).toBeFocused()
})

test('ordinary and sync notifications share one responsive surface contract', async ({ page }) => {
  const toastSource = readFileSync('src/components/common/Toast.vue', 'utf8')
  const syncToastSource = readFileSync('src/components/common/SyncTaskToast.vue', 'utf8')
  expect(toastSource).toContain('notification-surface')
  expect(syncToastSource).toContain('notification-surface')

  await page.setViewportSize({ width: 375, height: 667 })
  await page.evaluate(() => {
    const fixtures = [
      ['ordinary-notification-fixture', 'toast notification-surface'],
      ['sync-notification-fixture', 'toast-mode notification-surface'],
    ]

    for (const [id, className] of fixtures) {
      const surface = document.createElement('div')
      surface.id = id
      surface.className = className
      surface.textContent = 'A deliberately long notification message that must remain inside the narrow viewport.'
      surface.style.position = 'fixed'
      surface.style.right = '24px'
      surface.style.bottom = id.startsWith('ordinary') ? '24px' : '96px'
      document.body.append(surface)
    }
  })

  const ordinary = page.locator('#ordinary-notification-fixture')
  const sync = page.locator('#sync-notification-fixture')
  const properties = ['padding', 'borderRadius', 'borderLeftWidth', 'boxShadow', 'maxWidth'] as const

  const ordinaryStyles = await ordinary.evaluate((element, names) => {
    const styles = getComputedStyle(element)
    return Object.fromEntries(names.map((name) => [name, styles[name]]))
  }, properties)
  const syncStyles = await sync.evaluate((element, names) => {
    const styles = getComputedStyle(element)
    return Object.fromEntries(names.map((name) => [name, styles[name]]))
  }, properties)

  expect(syncStyles).toEqual(ordinaryStyles)
  expect(ordinaryStyles).toMatchObject({
    padding: '12px 16px',
    borderRadius: '8px',
    borderLeftWidth: '4px',
    maxWidth: '320px',
  })
  expect(ordinaryStyles.boxShadow).not.toBe('none')

  for (const surface of [ordinary, sync]) {
    const box = await surface.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.x).toBeGreaterThanOrEqual(24)
    expect(box!.x + box!.width).toBeLessThanOrEqual(351)
  }
})
