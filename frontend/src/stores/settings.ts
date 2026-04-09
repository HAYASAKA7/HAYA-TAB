import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Settings } from '@/types'
import { setLocale, type Locale } from '@/i18n'
import { SettingsService, TabService, CloudService } from '@/services'
import { Events } from '@wailsio/runtime'

export const useSettingsStore = defineStore('settings', () => {
  // State
  const settings = ref<Settings>({
    theme: 'system',
    language: 'en',
    background: '',
    bgType: '',
    openMethod: 'inner',
    openGpMethod: 'inner',
    audioDevice: 'default',
    syncPaths: [],
    syncStrategy: 'skip',
    autoSyncEnabled: false,
    autoSyncFrequency: 'startup',
    lastSyncTime: 0,
    keyBindings: {
      scrollDown: 'j',
      scrollUp: 'k',
      metronome: 'm',
      playPause: 'p',
      stop: 'o',
      bpmPlus: 'l',
      bpmMinus: 'h',
      toggleLoop: 'r',
      clearSelection: 'escape',
      jumpToBar: 't',
      jumpToStart: 'i',
      autoScroll: 'n',
      scrollSpeedUp: '.',
      scrollSpeedDown: ','
    },
    midiSettings: {
      enabled: false,
      scrollDown: null,
      scrollUp: null,
      playPause: null,
      expressionScroll: null
    },
    storagePath: '',
    coversPath: '',
    updateCheckEnabled: true,
    lastUpdateCheckTime: 0,
    latestVersion: '',
    webdavEnabled: false,
    webdavUrl: '',
    webdavUser: '',
    webdavPassword: ''
  })

  const loading = ref(false)
  const webdavConnected = ref(false)
  let webdavCheckInterval: ReturnType<typeof setInterval> | null = null
  let webdavSyncProgressUnregister: (() => void) | null = null

  // Actions
  async function loadSettings() {
    loading.value = true
    try {
      const loaded = await SettingsService.getSettings()
      if (loaded) {
        settings.value = {
          ...settings.value,
          ...loaded,
          language: loaded.language || 'en',
          audioDevice: loaded.audioDevice || 'default',
          syncPaths: loaded.syncPaths || [],
          keyBindings: {
            ...settings.value.keyBindings,
            ...(loaded.keyBindings || {})
          },
          midiSettings: {
            ...settings.value.midiSettings,
            ...(loaded.midiSettings || {})
          }
        }
      }
      applyTheme()
      applyLanguage()
      await applyBackground()
    } catch (err) {
      console.error('Error loading settings:', err)
    } finally {
      loading.value = false
    }
  }

  async function saveSettings() {
    loading.value = true
    try {
      await SettingsService.saveSettings(settings.value)
      applyTheme()
      applyLanguage()
      await applyBackground()

      // Start or stop WebDAV status check based on webdavEnabled
      if (settings.value.webdavEnabled) {
        startWebDAVStatusCheck()
      } else {
        stopWebDAVStatusCheck()
        webdavConnected.value = false
      }
    } catch (err) {
      console.error('Error saving settings:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  function applyTheme() {
    const theme = settings.value.theme
    if (theme === 'light') {
      document.body.setAttribute('data-theme', 'light')
    } else if (theme === 'dark') {
      document.body.removeAttribute('data-theme')
    } else {
      // System preference
      if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
        document.body.setAttribute('data-theme', 'light')
      } else {
        document.body.removeAttribute('data-theme')
      }
    }
  }

  function applyLanguage() {
    setLocale(settings.value.language as Locale)
  }

  async function applyBackground() {
    const layout = document.getElementById('app-layout')
    if (!layout) return

    if (settings.value.background && settings.value.bgType) {
      let bgUrl = settings.value.background

      if (settings.value.bgType === 'local' && !bgUrl.startsWith('http')) {
        try {
          const b64 = await TabService.getCover(bgUrl)
          if (b64) {
            bgUrl = `data:image/jpeg;base64,${b64}`
          }
        } catch (e) {
          console.error('Failed to load background:', e)
        }
      }

      layout.style.backgroundImage = `url('${bgUrl}')`
    } else {
      layout.style.backgroundImage = 'none'
    }
  }

  function addSyncPath(path: string) {
    if (!settings.value.syncPaths.includes(path)) {
      settings.value.syncPaths.push(path)
    }
  }

  function removeSyncPath(index: number) {
    settings.value.syncPaths.splice(index, 1)
  }

  async function triggerSync() {
    await SettingsService.saveSettings(settings.value)
    return await SettingsService.triggerSync()
  }

  // WebDAV connection status check
  async function checkWebDAVStatus() {
    if (!settings.value.webdavEnabled) {
      webdavConnected.value = false
      return false
    }

    // Also check browser online status
    if (!navigator.onLine) {
      webdavConnected.value = false
      return false
    }

    try {
      const connected = await CloudService.checkStatus()
      webdavConnected.value = connected
      return connected
    } catch (err) {
      console.error('WebDAV status check failed:', err)
      webdavConnected.value = false
      return false
    }
  }

  // Start periodic WebDAV status checks
  function startWebDAVStatusCheck(intervalMs = 30000) {
    stopWebDAVStatusCheck()
    checkWebDAVStatus() // Initial check
    webdavCheckInterval = setInterval(checkWebDAVStatus, intervalMs)

    // Listen for online/offline events
    window.addEventListener('online', checkWebDAVStatus)
    window.addEventListener('offline', () => {
      webdavConnected.value = false
    })

    // Listen for volume initialization completion to update status immediately
    webdavSyncProgressUnregister = Events.On('webdav-sync-progress', (ev: any) => {
      const data = ev.data ? (Array.isArray(ev.data) ? ev.data[0] : ev.data) : ev
      if (data.status === 'success') {
        webdavConnected.value = true
      } else if (data.status === 'error') {
        webdavConnected.value = false
      }
    })
  }

  // Stop periodic WebDAV status checks
  function stopWebDAVStatusCheck() {
    if (webdavCheckInterval) {
      clearInterval(webdavCheckInterval)
      webdavCheckInterval = null
    }
    if (webdavSyncProgressUnregister) {
      webdavSyncProgressUnregister()
      webdavSyncProgressUnregister = null
    }
  }

  // Watch for system theme changes
  if (window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', () => {
      if (settings.value.theme === 'system') {
        applyTheme()
      }
    })
  }

  return {
    settings,
    loading,
    webdavConnected,
    loadSettings,
    saveSettings,
    applyTheme,
    applyLanguage,
    applyBackground,
    addSyncPath,
    removeSyncPath,
    triggerSync,
    checkWebDAVStatus,
    startWebDAVStatusCheck,
    stopWebDAVStatusCheck
  }
})
