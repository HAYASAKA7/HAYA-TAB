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

    const horizontalOverflows = await dialog.evaluate((surface) => {
      const surfaceRect = surface.getBoundingClientRect()
      return Array.from(surface.querySelectorAll<HTMLElement>('*')).flatMap((element) => {
        const style = getComputedStyle(element)
        const rect = element.getBoundingClientRect()
        if (style.display === 'none' || style.visibility === 'hidden' || rect.width === 0) return []
        if (rect.left >= surfaceRect.left - 0.5 && rect.right <= surfaceRect.right + 0.5) return []
        return [`${element.tagName.toLowerCase()}.${element.className}: ${rect.left}-${rect.right}`]
      })
    })
    expect(horizontalOverflows, `${dialogName} descendant horizontal overflow`).toEqual([])

    const footer = dialog.locator('.modal-actions')
    if (await footer.count()) {
      const footerBox = await footer.boundingBox()
      expect(footerBox).not.toBeNull()
      expect(footerBox!.x).toBeGreaterThanOrEqual(box!.x)
      expect(footerBox!.x + footerBox!.width).toBeLessThanOrEqual(box!.x + box!.width)
      expect(footerBox!.y + footerBox!.height).toBeLessThanOrEqual(box!.y + box!.height)
    }
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

test('dialog keyboard events do not reach application shortcuts behind it', async ({ page }) => {
  await page.evaluate(() => {
    ;(window as any).__backgroundKeydownCount = 0
    window.addEventListener('keydown', () => {
      ;(window as any).__backgroundKeydownCount += 1
    })
  })

  await openDialog(page, 'keyBindings')
  await expect(page.getByTestId('base-modal')).toBeFocused()
  await page.keyboard.press('x')

  expect(await page.evaluate(() => (window as any).__backgroundKeydownCount)).toBe(0)
})

test('ordinary and sync notifications share one responsive surface contract', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 667 })
  await page.evaluate(() => {
    const app = document.querySelector('#app') as any
    app.__vue_app__._instance.setupState.showToast(
      'A deliberately long notification message that must remain inside the narrow viewport.',
      'error',
    )

    const dispatch = (window as any)._wails.dispatchWailsEvent
    dispatch({
      name: 'webdav-sync-progress',
      data: {
        status: 'error',
        message: 'A deliberately long sync notification that must remain inside the narrow viewport.',
      },
    })
  })

  const ordinary = page.locator('#toast-container .toast.notification-surface').first()
  const sync = page.locator('#sync-task-toast-container .toast-mode.notification-surface')
  await expect(ordinary).toBeVisible()
  await expect(sync).toBeVisible()
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
