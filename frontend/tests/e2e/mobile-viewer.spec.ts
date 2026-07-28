import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    const messages: unknown[] = []
    Object.defineProperty(window, '__nativeMessages', {
      value: messages,
      configurable: false
    })
    Object.defineProperty(window, 'webkit', {
      value: {
        messageHandlers: {
          hayaBridge: {
            postMessage(message: unknown) {
              messages.push(message)
              return Promise.resolve({ ok: true })
            }
          }
        }
      },
      configurable: false
    })
  })
  await page.goto('/mobile-viewer/')
})

test('announces a versioned ready message', async ({ page }) => {
  await expect.poll(async () =>
    page.evaluate(() => {
      const messages = (window as unknown as {
        __nativeMessages: Array<{ version?: number; type?: string }>
      }).__nativeMessages
      return messages.at(-1)
    })
  ).toEqual({ version: 1, type: 'ready' })
})

test('allow-lists commands and rejects unknown commands', async ({ page }) => {
  const supported = await page.evaluate(async () => {
    const viewer = (window as unknown as {
      hayaTabViewer: {
        receive(command: unknown): Promise<{ ok: boolean; error?: { code: string } }>
      }
    }).hayaTabViewer

    return Promise.all([
      viewer.receive({
        version: 1,
        type: 'load',
        payload: { base64: 'RjAw' }
      }),
      viewer.receive({ version: 1, type: 'playPause' }),
      viewer.receive({ version: 1, type: 'stop' }),
      viewer.receive({
        version: 1,
        type: 'setTempo',
        payload: { beatsPerMinute: 120 }
      }),
      viewer.receive({ version: 1, type: 'deleteEverything' })
    ])
  })

  expect(supported.slice(0, 4).every(result => result.ok)).toBe(true)
  expect(supported[4]).toEqual({
    ok: false,
    error: { code: 'unknown_command' }
  })
})

test('rejects unsupported bridge versions and invalid tempo', async ({ page }) => {
  const results = await page.evaluate(async () => {
    const viewer = (window as unknown as {
      hayaTabViewer: {
        receive(command: unknown): Promise<{ ok: boolean; error?: { code: string } }>
      }
    }).hayaTabViewer
    return Promise.all([
      viewer.receive({ version: 2, type: 'stop' }),
      viewer.receive({
        version: 1,
        type: 'setTempo',
        payload: { beatsPerMinute: 999 }
      })
    ])
  })

  expect(results).toEqual([
    { ok: false, error: { code: 'unsupported_version' } },
    { ok: false, error: { code: 'invalid_payload' } }
  ])
})
