<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { SUPPORTED_LOCALES } from '@/i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const uiStore = useUIStore()
const { showToast } = useToast()
const audioDevices = ref<MediaDeviceInfo[]>([])
const isAudioOutputSupported = ref(false)
const syncStatus = ref('')
const syncFilename = ref('')
const syncCount = ref(0)
const isSyncing = ref(false)

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
    showToast(t('toast.failedAudioDevices') + ': ' + e, 'error')
  }
}

async function handleSave() {
  try {
    await settingsStore.saveSettings()
    showToast(t('settings.settingsSaved'))
  } catch (err) {
    showToast(t('settings.errorSaving') + ': ' + err, 'error')
  }
}

async function handleAddSyncPath() {
  const path = await window.go.main.App.SelectFolder()
  if (path) {
    settingsStore.addSyncPath(path)
  }
}

async function handleBrowseBg() {
  const path = await window.go.main.App.SelectImage()
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

  EventsOn('sync-progress', (data: any) => {
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
    showToast(t('toast.syncError') + ': ' + err, 'error')
    syncStatus.value = t('settings.syncFailed')
  } finally {
    EventsOff('sync-progress')
    isSyncing.value = false
    setTimeout(() => {
      if (syncStatus.value === t('settings.syncCompleted')) {
        syncStatus.value = ''
        syncCount.value = 0
      }
    }, 3000)
  }
}
</script>

<template>
  <header><h1>{{ t('settings.title') }}</h1></header>
  <div class="settings-container">
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

    <section class="settings-section">
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
            <span>{{ path }}</span>
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

    <div class="settings-footer">
      <button class="btn primary" @click="handleSave">{{ t('settings.saveChanges') }}</button>
    </div>
  </div>
</template>
