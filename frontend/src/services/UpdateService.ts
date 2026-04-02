import { useSettingsStore } from '@/stores/settings'

export interface UpdateInfo {
  hasUpdate: boolean
  latestVersion: string
  currentVersion: string
  releaseUrl: string
}

export const UpdateService = {
  async checkForUpdates(force = false): Promise<UpdateInfo | null> {
    try {
      const settingsStore = useSettingsStore()
      
      // If auto check is disabled and this is not a forced check, return null
      if (!settingsStore.settings.updateCheckEnabled && !force) {
        return null
      }

      const now = Date.now()
      const cooldown = 12 * 60 * 60 * 1000 // 12 hours in milliseconds

      // Use cached check result if within cooldown period, unless forced
      if (!force && 
          settingsStore.settings.lastUpdateCheckTime && 
          now - settingsStore.settings.lastUpdateCheckTime < cooldown) {
        
        const currentVersion = await this.getCurrentVersion()
        if (settingsStore.settings.latestVersion && settingsStore.settings.latestVersion !== currentVersion) {
          const hasUpdate = this.compareVersions(settingsStore.settings.latestVersion, currentVersion) > 0
          return {
            hasUpdate,
            latestVersion: settingsStore.settings.latestVersion,
            currentVersion,
            releaseUrl: 'https://github.com/HAYASAKA7/HAYA-TAB/releases/latest'
          }
        }
        return null
      }

      // Fetch from GitHub API
      const response = await fetch('https://api.github.com/repos/HAYASAKA7/HAYA-TAB/releases/latest')
      if (!response.ok) {
        throw new Error('Failed to fetch latest release')
      }

      const data = await response.json()
      let latestVersion = data.tag_name || ''
      if (latestVersion.startsWith('v')) {
        latestVersion = latestVersion.substring(1)
      }

      const currentVersion = await this.getCurrentVersion()
      
      // Save last check time and latest version
      settingsStore.settings.lastUpdateCheckTime = now
      settingsStore.settings.latestVersion = latestVersion
      await settingsStore.saveSettings()

      const hasUpdate = this.compareVersions(latestVersion, currentVersion) > 0

      return {
        hasUpdate,
        latestVersion,
        currentVersion,
        releaseUrl: data.html_url || 'https://github.com/HAYASAKA7/HAYA-TAB/releases/latest'
      }

    } catch (error) {
      console.error('Error checking for updates:', error)
      return null
    }
  },

  async getCurrentVersion(): Promise<string> {
    try {
      // @ts-ignore
      if (window['go'] && window['go']['main'] && window['go']['main']['App'] && window['go']['main']['App']['GetAppVersion']) {
        // @ts-ignore
        return await window['go']['main']['App']['GetAppVersion']()
      }
      return '2.4.20' // Fallback
    } catch (e) {
      console.warn('Failed to get app version from backend', e)
      return '2.4.20'
    }
  },

  // Returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
  compareVersions(v1: string, v2: string): number {
    const parts1 = v1.replace(/^v/, '').split('.').map(Number)
    const parts2 = v2.replace(/^v/, '').split('.').map(Number)
    
    const len = Math.max(parts1.length, parts2.length)
    for (let i = 0; i < len; i++) {
      const num1 = parts1[i] || 0
      const num2 = parts2[i] || 0
      
      if (num1 > num2) return 1
      if (num1 < num2) return -1
    }
    return 0
  }
}
