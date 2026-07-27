<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore, useUIStore, usePlatformStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { FileService, SettingsService, UpdateService } from '@/services'
import { midiService } from '@/services/MidiService'
import { Events } from "@wailsio/runtime"
import { SUPPORTED_LOCALES } from '@/i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const uiStore = useUIStore()
const platformStore = usePlatformStore()
const { showToast, showErrorToast } = useToast()
const audioDevices = ref<MediaDeviceInfo[]>([])
const isAudioOutputSupported = ref(false)
const syncStatus = ref('')
const syncFilename = ref('')
const syncCount = ref(0)
const isSyncing = ref(false)
const isMigrating = ref(false)
const isCheckingUpdate = ref(false)

// MIDI Learn state
const learningAction = ref<string | null>(null)

function startMidiLearn(action: string) {
  learningAction.value = action
  midiService.enterLearnMode((mapping) => {
    // Update the corresponding MIDI mapping based on the action
    if (learningAction.value === 'scrollDown') settingsStore.settings.midiSettings.scrollDown = mapping
    else if (learningAction.value === 'scrollUp') settingsStore.settings.midiSettings.scrollUp = mapping
    else if (learningAction.value === 'playPause') settingsStore.settings.midiSettings.playPause = mapping
    else if (learningAction.value === 'expressionScroll') settingsStore.settings.midiSettings.expressionScroll = mapping
    
    learningAction.value = null
    showToast(t('settings.midiMapped', 'MIDI Mapping Successful'), 'success')
  })
}

function cancelMidiLearn() {
  learningAction.value = null
  midiService.cancelLearnMode()
}

function formatMidiMapping(mapping: any) {
  if (!mapping) return t('settings.notMapped', 'Not Mapped')
  return `${mapping.type} ${mapping.number} (Ch ${mapping.channel + 1})`
}

async function handleCheckUpdate() {
  if (isCheckingUpdate.value) return
  isCheckingUpdate.value = true
  try {
    const info = await UpdateService.checkForUpdates(true)
    if (info) {
      uiStore.updateInfo = info
      if (info.hasUpdate) {
        showToast(t('settings.updateAvailable'), 'info')
      } else {
        showToast(t('settings.upToDate'), 'success')
      }
    } else {
      showToast(t('errors.network', { operation: 'update check' }), 'error')
    }
  } catch (e) {
    console.error(e)
    showToast(t('errors.unknown'), 'error')
  } finally {
    isCheckingUpdate.value = false
  }
}

// Auto-save when settings change (excluding keyBindings which saves on modal close)
watch(
  () => ({
    theme: settingsStore.settings.theme,
    language: settingsStore.settings.language,
    background: settingsStore.settings.background,
    bgType: settingsStore.settings.bgType,
    openMethod: settingsStore.settings.openMethod,
    openGpMethod: settingsStore.settings.openGpMethod,
    audioDevice: settingsStore.settings.audioDevice,
    syncPaths: [...settingsStore.settings.syncPaths],
    syncStrategy: settingsStore.settings.syncStrategy,
    autoSyncEnabled: settingsStore.settings.autoSyncEnabled,
    autoSyncFrequency: settingsStore.settings.autoSyncFrequency,
    updateCheckEnabled: settingsStore.settings.updateCheckEnabled,
    webdavEnabled: settingsStore.settings.webdavEnabled, // Only watch enabled state
    midiEnabled: settingsStore.settings.midiSettings.enabled,
    midiMappings: JSON.stringify(settingsStore.settings.midiSettings),
  }),
  async () => {
    try {
      await settingsStore.saveSettings()
    } catch (err) {
      showErrorToast(err, t('settings.errorSaving'))
    }
  },
  { deep: true }
)

onMounted(async () => {
  // Check if AudioContext supports setSinkId (required for changing output device)
  // @ts-ignore
  if (window.AudioContext && typeof AudioContext.prototype.setSinkId === 'function') {
    isAudioOutputSupported.value = true
    await fetchAudioDevices()
  }
})

async function fetchAudioDevices() {
  try {
    if (!navigator.mediaDevices || !navigator.mediaDevices.enumerateDevices) {
      console.warn('Media devices API not supported')
      return
    }

    // First try without requesting permission
    let devices = await navigator.mediaDevices.enumerateDevices()

    // Check if we have audio output devices and if they have labels
    // If labels are empty, we might need permission
    const hasAudioOutput = devices.some(d => d.kind === 'audiooutput')
    const hasLabels = devices.some(d => d.kind === 'audiooutput' && d.label)

    if (hasAudioOutput && !hasLabels) {
      console.log('Audio devices found but execution blocked/no labels. Requesting permission...')
      try {
        // Request microphone permission to reveal device labels/ids
        // This is a browser security restriction
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
        // Stop the stream immediately, we only needed permission
        stream.getTracks().forEach(t => t.stop())

        // Refresh valid devices list
        devices = await navigator.mediaDevices.enumerateDevices()
      } catch (permErr) {
        console.warn('Permission denied for audio devices or no microphone found:', permErr)
        // Continue with what we have (ids might still work even if labels are empty, though less useful)
      }
    }

    audioDevices.value = devices.filter(d => d.kind === 'audiooutput')
  } catch (e) {
    console.error('Error fetching audio devices', e)
    showToast(t('toast.failedAudioDevices'), 'error')
  }
}

async function handleAddSyncPath() {
  const path = await FileService.selectFolder()
  if (path) {
    settingsStore.addSyncPath(path)
  }
}

async function handleBrowseBg() {
  const path = await FileService.selectImage()
  if (path) {
    settingsStore.settings.background = path
  }
}

async function handleSync() {
  if (isSyncing.value) return
  isSyncing.value = true
  syncStatus.value = t('settings.startingSync')
  syncFilename.value = ''
  syncCount.value = 0

  const unregisterSync = Events.On('sync-progress', (ev: any) => {
    const data = ev.data ? (Array.isArray(ev.data) ? ev.data[0] : ev.data) : ev
    if (data) {
      if (data.message) {
        syncStatus.value = data.message
      }
      if (data.count !== undefined) {
        syncCount.value = data.count
      }
    }
  })

  try {
    const msg = await settingsStore.triggerSync()
    showToast(msg)
    syncStatus.value = t('settings.syncCompleted')
  } catch (err) {
    showErrorToast(err, t('toast.syncError'))
    syncStatus.value = t('settings.syncFailed')
  } finally {
    if (unregisterSync) unregisterSync()
    isSyncing.value = false
    setTimeout(() => {
      if (syncStatus.value === t('settings.syncCompleted')) {
        syncStatus.value = ''
        syncCount.value = 0
      }
    }, 3000)
  }
}

function handleWebDAVToggle() {
  if (settingsStore.settings.webdavEnabled) {
    // If enabled, check if URL is set. If not, open modal.
    if (!settingsStore.settings.webdavUrl) {
      uiStore.showWebdavModal()
    } else {
      settingsStore.startWebDAVStatusCheck()
    }
  } else {
    settingsStore.stopWebDAVStatusCheck()
  }
}

async function handleChangePath(target: 'storage' | 'covers') {
  if (isMigrating.value) return
  const currentPath = target === 'storage' ? settingsStore.settings.storagePath : settingsStore.settings.coversPath
  let selectedPath = await FileService.selectFolder()
  if (!selectedPath) return
  
  // Clean up path by removing trailing slashes
  selectedPath = selectedPath.replace(/[/\\]$/, '')
  const separator = selectedPath.includes('\\') ? '\\' : '/'
  
  // Ensure we append HAYA-TAB/target, avoiding duplicates if they selected the exact folder
  let targetSuffix = 'HAYA-TAB' + separator + target
  let newPath = selectedPath
  
  if (!selectedPath.endsWith(targetSuffix)) {
    // Check if they selected a folder ending in HAYA-TAB
    if (selectedPath.endsWith('HAYA-TAB')) {
      newPath = selectedPath + separator + target
    } else {
      newPath = selectedPath + separator + targetSuffix
    }
  }

  if (newPath === currentPath) return

  isMigrating.value = true
  try {
    const status = await SettingsService.checkMigration(target)
    const count = status.count
    const size = status.size
    if (count > 0) {
      const sizeMB = (size / 1024 / 1024).toFixed(2)
      uiStore.showConfirmModal(
        t('settings.migrateTitle', 'Directory Changed'),
        t('settings.migrateMsg', `The selected directory has changed. There are ${count} files (${sizeMB} MB) in the current directory.<br><br>Do you want to MIGRATE all existing files to the new directory, or ONLY APPLY the new path (leaving old files where they are)?`),
        t('settings.migrateBtn', 'Migrate All Files'),
        false,
        async () => {
          // Migrate
          try {
            showToast(t('settings.migrating', 'Migrating data, please wait...'), 'info')
            await SettingsService.migrateData(target, newPath, false)
            if (target === 'storage') settingsStore.settings.storagePath = newPath
            else settingsStore.settings.coversPath = newPath
            await settingsStore.saveSettings()
            showToast(t('settings.migrateSuccess', 'Migration completed successfully'), 'success')
          } catch (e) {
            showErrorToast(e)
          } finally {
            isMigrating.value = false
          }
        },
        t('settings.applyBtn', 'Only Apply Path'),
        async () => {
          // Only Apply
          try {
            await SettingsService.migrateData(target, newPath, true)
            if (target === 'storage') settingsStore.settings.storagePath = newPath
            else settingsStore.settings.coversPath = newPath
            await settingsStore.saveSettings()
            showToast(t('settings.pathApplied', 'Path applied successfully'), 'success')
          } catch (e) {
            showErrorToast(e)
          } finally {
            isMigrating.value = false
          }
        }
      )
    } else {
      // No files to migrate
      if (target === 'storage') settingsStore.settings.storagePath = newPath
      else settingsStore.settings.coversPath = newPath
      await settingsStore.saveSettings()
      showToast(t('settings.pathApplied', 'Path applied successfully'), 'success')
      isMigrating.value = false
    }
  } catch (err) {
    showErrorToast(err)
    isMigrating.value = false
  }
}
</script>

<template>
  <header><h1>{{ t('settings.title') }}</h1></header>
  <div class="settings-container">
    <section
      v-if="platformStore.capabilities.customStoragePaths"
      class="settings-section"
      data-testid="custom-storage-settings"
    >
      <h3><span class="icon-folder"></span> {{ t('settings.dataStorage', 'Data Storage') }}</h3>
      <div class="form-group">
        <label>{{ t('settings.storagePath', 'Managed Tabs Path') }}</label>
        <div class="input-with-button">
          <input type="text" :value="settingsStore.settings.storagePath || t('settings.defaultPath', 'Default (User Directory)')" disabled readonly>
          <button class="btn" @click="handleChangePath('storage')" :disabled="isMigrating">{{ t('settings.change', 'Change') }}</button>
        </div>
      </div>
      <div class="form-group">
        <label>{{ t('settings.coversPath', 'Covers Path') }}</label>
        <div class="input-with-button">
          <input type="text" :value="settingsStore.settings.coversPath || t('settings.defaultPath', 'Default (User Directory)')" disabled readonly>
          <button class="btn" @click="handleChangePath('covers')" :disabled="isMigrating">{{ t('settings.change', 'Change') }}</button>
        </div>
      </div>
    </section>

    <section class="settings-section">
      <h3><span class="icon-palette"></span> {{ t('settings.appearance') }}</h3>
      <div class="form-group">
        <label>{{ t('settings.language') }}</label>
        <select v-model="settingsStore.settings.language">
          <option v-for="locale in SUPPORTED_LOCALES" :key="locale.value" :value="locale.value">
            {{ locale.label }}
          </option>
        </select>
      </div>
      <div class="form-group">
        <label>{{ t('settings.theme') }}</label>
        <select id="set-theme" v-model="settingsStore.settings.theme">
          <option value="system">{{ t('settings.themeSystem') }}</option>
          <option value="dark">{{ t('settings.themeDark') }}</option>
          <option value="light">{{ t('settings.themeLight') }}</option>
        </select>
      </div>
      <div class="form-group">
        <label>{{ t('settings.backgroundImage') }}</label>
        <select id="set-bg-type" v-model="settingsStore.settings.bgType">
          <option value="">{{ t('settings.bgNone') }}</option>
          <option value="url">{{ t('settings.bgUrl') }}</option>
          <option value="local">{{ t('settings.bgLocal') }}</option>
        </select>
      </div>
      <div
        v-if="settingsStore.settings.bgType"
        id="bg-input-wrapper"
        class="form-group"
      >
        <input
          type="text"
          id="set-bg-val"
          v-model="settingsStore.settings.background"
          :placeholder="t('settings.bgPlaceholder')"
        />
        <button
          v-if="settingsStore.settings.bgType === 'local'"
          class="btn"
          id="btn-browse-bg"
          @click="handleBrowseBg"
        >
          {{ t('settings.browse') }}
        </button>
      </div>
    </section>

    <section class="settings-section">
      <h3><span class="icon-document"></span> {{ t('settings.viewers') }}</h3>
      <div class="form-group">
        <label>{{ t('settings.openPdfMethod') }}</label>
        <div class="radio-group">
          <label>
            <input
              type="radio"
              name="openMethod"
              value="system"
              v-model="settingsStore.settings.openMethod"
            />
            {{ t('settings.systemApp') }}
          </label>
          <label>
            <input
              type="radio"
              name="openMethod"
              value="inner"
              v-model="settingsStore.settings.openMethod"
            />
            {{ t('settings.builtInViewer') }}
          </label>
        </div>
      </div>
      <div class="form-group">
        <label>{{ t('settings.openGpMethod') }}</label>
        <div class="radio-group">
          <label>
            <input
              type="radio"
              name="openGpMethod"
              value="system"
              v-model="settingsStore.settings.openGpMethod"
            />
            {{ t('settings.systemApp') }}
          </label>
          <label>
            <input
              type="radio"
              name="openGpMethod"
              value="inner"
              v-model="settingsStore.settings.openGpMethod"
            />
            {{ t('settings.builtInAlphaTab') }}
          </label>
        </div>
      </div>
    </section>

    <section class="settings-section" v-if="isAudioOutputSupported">
      <h3><span class="icon-volume"></span> {{ t('settings.audio') }}</h3>
      <div class="form-group">
        <label>{{ t('settings.outputDevice') }}</label>
        <select v-model="settingsStore.settings.audioDevice">
          <option value="default">{{ t('settings.default') }}</option>
          <option
            v-for="device in audioDevices"
            :key="device.deviceId"
            :value="device.deviceId"
          >
            {{ device.label || t('settings.unknownDevice') + ' (' + device.deviceId.slice(0, 8) + '...)' }}
          </option>
        </select>
        <p class="hint">{{ t('settings.audioHint') }}</p>
      </div>
    </section>

    <section class="settings-section">
      <h3><span class="icon-keyboard"></span> {{ t('settings.shortcuts') }}</h3>
      <div class="form-group">
        <label>{{ t('settings.keyBindings') }}</label>
        <button class="btn" @click="uiStore.showKeyBindingsModal">{{ t('settings.configureKeyBindings') }}</button>
      </div>
    </section>

    <section v-if="platformStore.capabilities.webMIDI" class="settings-section">
      <h3><span class="icon-piano"></span> MIDI</h3>
      <div class="form-group">
        <label>
          <input type="checkbox" v-model="settingsStore.settings.midiSettings.enabled">
          {{ t('settings.enableMidi', 'Enable MIDI Pedal Support') }}
        </label>
      </div>
      <div v-if="settingsStore.settings.midiSettings.enabled" class="midi-config">
        <div class="form-group">
          <label>{{ t('settings.midiScrollDown', 'Scroll Down / Next Page') }}</label>
          <div class="input-with-button">
            <input type="text" :value="formatMidiMapping(settingsStore.settings.midiSettings.scrollDown)" disabled readonly>
            <button class="btn" @click="startMidiLearn('scrollDown')" :class="{ 'primary': learningAction === 'scrollDown' }">
              {{ learningAction === 'scrollDown' ? t('settings.learning', 'Waiting for signal...') : t('settings.learn', 'Learn') }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>{{ t('settings.midiScrollUp', 'Scroll Up / Previous Page') }}</label>
          <div class="input-with-button">
            <input type="text" :value="formatMidiMapping(settingsStore.settings.midiSettings.scrollUp)" disabled readonly>
            <button class="btn" @click="startMidiLearn('scrollUp')" :class="{ 'primary': learningAction === 'scrollUp' }">
              {{ learningAction === 'scrollUp' ? t('settings.learning', 'Waiting for signal...') : t('settings.learn', 'Learn') }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>{{ t('settings.midiPlayPause', 'Play / Pause') }}</label>
          <div class="input-with-button">
            <input type="text" :value="formatMidiMapping(settingsStore.settings.midiSettings.playPause)" disabled readonly>
            <button class="btn" @click="startMidiLearn('playPause')" :class="{ 'primary': learningAction === 'playPause' }">
              {{ learningAction === 'playPause' ? t('settings.learning', 'Waiting for signal...') : t('settings.learn', 'Learn') }}
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>{{ t('settings.midiExpression', 'Expression Pedal (Smooth Scroll)') }}</label>
          <div class="input-with-button">
            <input type="text" :value="formatMidiMapping(settingsStore.settings.midiSettings.expressionScroll)" disabled readonly>
            <button class="btn" @click="startMidiLearn('expressionScroll')" :class="{ 'primary': learningAction === 'expressionScroll' }">
              {{ learningAction === 'expressionScroll' ? t('settings.learning', 'Waiting for signal...') : t('settings.learn', 'Learn') }}
            </button>
          </div>
        </div>
        <div v-if="learningAction" class="midi-learn-overlay">
          <p>{{ t('settings.midiLearnTip', 'Please press or step on your MIDI device...') }}</p>
          <button class="btn small" @click="cancelMidiLearn">{{ t('common.cancel', 'Cancel') }}</button>
        </div>
      </div>
    </section>

    <section class="settings-section">
      <h3><span class="icon-cloud"></span> WebDAV</h3>
      <div class="form-group">
        <label>
          <input type="checkbox" v-model="settingsStore.settings.webdavEnabled" @change="handleWebDAVToggle">
          {{ t('settings.enableWebdav') }}
        </label>
      </div>
      <div v-if="settingsStore.settings.webdavEnabled" class="webdav-info">
        <div class="form-group">
          <label>{{ t('settings.webdavAddress') }}</label>
          <div class="input-with-button">
            <input type="text" :value="settingsStore.settings.webdavUrl" disabled readonly>
            <button class="btn" @click="uiStore.showWebdavModal" :title="t('settings.configureWebdav')"><span class="icon-edit"></span></button>
            <button class="btn" :disabled="!settingsStore.webdavConnected" @click="uiStore.showCloudPickerModal" :title="settingsStore.webdavConnected ? t('cloud.title') : t('cloud.offline')"><span class="icon-cloud"></span></button>
          </div>
        </div>
      </div>
    </section>

    <section v-if="platformStore.capabilities.folderWatcher" class="settings-section">
      <h3><span class="icon-sync"></span> {{ t('settings.autoSync') }}</h3>
      <div class="form-group">
        <label>
          <input type="checkbox" v-model="settingsStore.settings.autoSyncEnabled">
          {{ t('settings.enableAutoSync') }}
        </label>
      </div>
      <div class="form-group" v-if="settingsStore.settings.autoSyncEnabled">
        <label>{{ t('settings.syncFrequency') }}</label>
        <select v-model="settingsStore.settings.autoSyncFrequency">
          <option value="startup">{{ t('settings.freqStartup') }}</option>
          <option value="weekly">{{ t('settings.freqWeekly') }}</option>
          <option value="monthly">{{ t('settings.freqMonthly') }}</option>
          <option value="yearly">{{ t('settings.freqYearly') }}</option>
        </select>
      </div>
      <div class="form-group">
        <label>{{ t('settings.syncStrategy') }}</label>
        <select id="set-sync-strategy" v-model="settingsStore.settings.syncStrategy">
          <option value="skip">{{ t('settings.strategySkip') }}</option>
          <option value="overwrite">{{ t('settings.strategyOverwrite') }}</option>
        </select>
      </div>
      <div class="form-group">
        <label>{{ t('settings.monitoredFolders') }}</label>
        <ul id="sync-path-list">
          <li v-for="(path, index) in settingsStore.settings.syncPaths" :key="index">
            <span class="selectable">{{ path }}</span>
            <span class="delete-icon" @click="settingsStore.removeSyncPath(index)">
              <span class="icon-trash"></span>
            </span>
          </li>
        </ul>
        <button class="btn small" @click="handleAddSyncPath">{{ t('settings.addFolder') }}</button>
      </div>
      <div class="sync-actions">
        <button class="btn primary" @click="handleSync" :disabled="isSyncing">
          <span v-if="isSyncing" class="sync-spinner"></span>
          {{ isSyncing ? t('settings.syncing') : t('settings.syncNow') }}
        </button>
        <div v-if="isSyncing || syncStatus" class="sync-progress-container">
          <div v-if="isSyncing" class="sync-progress-bar">
            <div class="sync-progress-bar-inner"></div>
          </div>
          <div class="sync-progress-info">
            <span class="sync-status">{{ syncStatus }}</span>
            <span v-if="syncCount > 0" class="sync-count">({{ syncCount }} {{ t('settings.filesProcessed') }})</span>
          </div>
        </div>
      </div>
    </section>

    <section
      v-if="platformStore.capabilities.selfUpdate"
      class="settings-section"
      data-testid="self-update-settings"
    >
      <h3><span class="icon-refresh"></span> {{ t('settings.update', 'Update') }}</h3>
      <div class="form-group">
        <label>
          <input type="checkbox" v-model="settingsStore.settings.updateCheckEnabled">
          {{ t('settings.enableUpdateCheck', 'Enable Update Check') }}
        </label>
      </div>
      <div class="form-group">
        <label>{{ t('settings.currentVersion', 'Current Version') }}</label>
        <div class="info-text">{{ uiStore.updateInfo?.currentVersion || settingsStore.settings.latestVersion || '3.1.7' }}</div>
      </div>
      <div class="form-group" v-if="uiStore.updateInfo?.hasUpdate">
        <label>{{ t('settings.latestVersion', 'Latest Version') }}</label>
        <div class="info-text update-available">{{ uiStore.updateInfo.latestVersion }} ({{ t('settings.updateAvailable', 'Update Available') }})</div>
      </div>
      <div class="form-group update-actions">
        <button class="btn primary" @click="handleCheckUpdate" :disabled="isCheckingUpdate">
          {{ isCheckingUpdate ? t('settings.syncing', 'Checking...') : t('settings.checkUpdates', 'Check for Updates') }}
        </button>
        <a 
          v-if="uiStore.updateInfo?.hasUpdate" 
          class="btn primary" 
          :href="uiStore.updateInfo.releaseUrl" 
          target="_blank"
          rel="noopener noreferrer"
          style="text-decoration: none; display: inline-flex; align-items: center;"
        >
          {{ t('settings.goToUpdate', 'Go to Update') }}
        </a>
      </div>
    </section>
  </div>
</template>

<style scoped>
.input-with-button {
  display: flex;
  gap: 0.5rem;
}

.input-with-button input {
  flex: 1;
}

.input-with-button .btn {
  padding: 0 0.75rem;
}

.midi-learn-overlay {
  margin-top: 1rem;
  padding: 1rem;
  background: var(--bg-secondary);
  border: 1px dashed var(--primary-color);
  border-radius: 4px;
  text-align: center;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% { opacity: 0.8; }
  50% { opacity: 1; }
  100% { opacity: 0.8; }
}
</style>
