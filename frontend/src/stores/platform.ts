import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { callBackend } from '@/services/api'
import type { RuntimeCapabilities } from '@/types/platform'

const safeFallback: RuntimeCapabilities = {
  target: 'desktop',
  formFactor: 'desktop',
  nativeTopLevelTabs: false,
  webTopLevelTabs: false,
  inProcessContent: false,
  loopbackContent: false,
  nativeFileImport: false,
  safeAreaInsets: false,
  folderWatcher: false,
  customStoragePaths: false,
  plugins: false,
  webMIDI: false,
  selfUpdate: false,
}

export const usePlatformStore = defineStore('platform', () => {
  const capabilities = ref<RuntimeCapabilities>({ ...safeFallback })
  const ready = ref(false)
  const isMobile = computed(() => capabilities.value.target !== 'desktop')

  async function load(viewportWidth = window.innerWidth) {
    const testCapabilities = import.meta.env.DEV
      ? window.__HAYA_TEST_CAPABILITIES__
      : undefined

    try {
      capabilities.value = testCapabilities
        ?? await callBackend<RuntimeCapabilities>(
          'GetRuntimeCapabilities',
          Math.max(0, Math.round(viewportWidth)),
        )
    } catch (error) {
      capabilities.value = { ...safeFallback }
      console.error('Failed to load runtime capabilities; using safe defaults:', error)
    } finally {
      document.documentElement.dataset.runtimeTarget = capabilities.value.target
      document.documentElement.dataset.formFactor = capabilities.value.formFactor
      ready.value = true
    }
  }

  return {
    capabilities,
    ready,
    isMobile,
    load,
  }
})
