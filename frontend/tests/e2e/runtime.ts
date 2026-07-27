import type { Page } from '@playwright/test'
import type { RuntimeCapabilities } from '../../src/types/platform'

export const iosCapabilities = {
  target: 'ios',
  formFactor: 'phone',
  nativeTopLevelTabs: true,
  webTopLevelTabs: false,
  inProcessContent: true,
  loopbackContent: false,
  nativeFileImport: true,
  safeAreaInsets: true,
  folderWatcher: false,
  customStoragePaths: false,
  plugins: false,
  webMIDI: false,
  selfUpdate: false,
} as const satisfies RuntimeCapabilities

export const desktopCapabilities = {
  target: 'desktop',
  formFactor: 'desktop',
  nativeTopLevelTabs: false,
  webTopLevelTabs: false,
  inProcessContent: false,
  loopbackContent: true,
  nativeFileImport: false,
  safeAreaInsets: false,
  folderWatcher: true,
  customStoragePaths: true,
  plugins: true,
  webMIDI: true,
  selfUpdate: true,
} as const satisfies RuntimeCapabilities

export const androidCapabilities = {
  target: 'android',
  formFactor: 'phone',
  nativeTopLevelTabs: false,
  webTopLevelTabs: true,
  inProcessContent: true,
  loopbackContent: false,
  nativeFileImport: true,
  safeAreaInsets: true,
  folderWatcher: false,
  customStoragePaths: false,
  plugins: false,
  webMIDI: false,
  selfUpdate: false,
} as const satisfies RuntimeCapabilities

export const iosTabletCapabilities = {
  ...iosCapabilities,
  formFactor: 'tablet',
} as const satisfies RuntimeCapabilities

export async function installRuntime(page: Page, capabilities: RuntimeCapabilities) {
  await page.addInitScript((runtimeCapabilities) => {
    const testWindow = window as typeof window & {
      __HAYA_TEST_CAPABILITIES__?: unknown
      __HAYA_MIDI_REQUESTS__?: number
    }

    testWindow.__HAYA_TEST_CAPABILITIES__ = runtimeCapabilities
    testWindow.__HAYA_MIDI_REQUESTS__ = 0

    Object.defineProperty(navigator, 'requestMIDIAccess', {
      configurable: true,
      value: async () => {
        testWindow.__HAYA_MIDI_REQUESTS__! += 1
        return {
          inputs: new Map(),
          onstatechange: null,
        }
      },
    })
  }, capabilities)
}

export async function installBackend(page: Page, responses: Record<string, unknown>) {
  await page.addInitScript((backendResponses) => {
    const testWindow = window as typeof window & {
      __HAYA_BACKEND_CALLS__?: Array<{ method: string; args: unknown[] }>
      __HAYA_TEST_BACKEND__?: Record<string, (...args: unknown[]) => unknown>
    }

    testWindow.__HAYA_BACKEND_CALLS__ = []
    testWindow.__HAYA_TEST_BACKEND__ = Object.fromEntries(
      Object.entries(backendResponses).map(([method, response]) => [
        method,
        (...args: unknown[]) => {
          testWindow.__HAYA_BACKEND_CALLS__!.push({ method, args })
          return response
        },
      ]),
    )
  }, responses)
}
