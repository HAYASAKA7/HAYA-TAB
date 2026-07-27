import { expect, test, type Page } from '@playwright/test'

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
