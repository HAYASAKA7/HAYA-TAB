import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Settings } from '@/types'

export const useSettingsStore = defineStore('settings', () => {
  // State
  const settings = ref<Settings>({
    theme: 'system',
    background: '',
    bgType: '',
    openMethod: 'inner',
    openGpMethod: 'inner',
    syncPaths: [],
    syncStrategy: 'skip'
  })

  const loading = ref(false)

  // Actions
  async function loadSettings() {
    loading.value = true
    try {
      const loaded = await window.go.main.App.GetSettings()
      if (loaded) {
        settings.value = {
          ...settings.value,
          ...loaded,
          syncPaths: loaded.syncPaths || []
        }
      }
      applyTheme()
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
      await window.go.main.App.SaveSettings(settings.value)
      applyTheme()
      await applyBackground()
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

  async function applyBackground() {
    const layout = document.getElementById('app-layout')
    if (!layout) return

    if (settings.value.background && settings.value.bgType) {
      let bgUrl = settings.value.background

      if (settings.value.bgType === 'local' && !bgUrl.startsWith('http')) {
        try {
          const b64 = await window.go.main.App.GetCover(bgUrl)
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
    await window.go.main.App.SaveSettings(settings.value)
    return await window.go.main.App.TriggerSync()
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
    loadSettings,
    saveSettings,
    applyTheme,
    applyBackground,
    addSyncPath,
    removeSyncPath,
    triggerSync
  }
})
