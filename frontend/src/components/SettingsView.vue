<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSettingsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'

const settingsStore = useSettingsStore()
const uiStore = useUIStore()
const { showToast } = useToast()
const audioDevices = ref<MediaDeviceInfo[]>([])
const isAudioOutputSupported = ref(false)

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
    showToast('Failed to list audio devices: ' + e, 'error')
  }
}

async function handleSave() {
  try {
    await settingsStore.saveSettings()
    showToast('Settings saved')
  } catch (err) {
    showToast('Error saving settings: ' + err, 'error')
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
  showToast('Sync started...')
  try {
    const msg = await settingsStore.triggerSync()
    showToast(msg)
  } catch (err) {
    showToast('Sync error: ' + err, 'error')
  }
}
</script>

<template>
  <header><h1>Settings</h1></header>
  <div class="settings-container">
    <section class="settings-section">
      <h3><span class="icon-palette"></span> Appearance</h3>
      <div class="form-group">
        <label>Theme</label>
        <select id="set-theme" v-model="settingsStore.settings.theme">
          <option value="system">System Default</option>
          <option value="dark">Dark</option>
          <option value="light">Light</option>
        </select>
      </div>
      <div class="form-group">
        <label>Background Image</label>
        <select id="set-bg-type" v-model="settingsStore.settings.bgType">
          <option value="">None</option>
          <option value="url">Online URL</option>
          <option value="local">Local File</option>
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
          placeholder="Enter URL or Path"
        />
        <button
          v-if="settingsStore.settings.bgType === 'local'"
          class="btn"
          id="btn-browse-bg"
          @click="handleBrowseBg"
        >
          Browse...
        </button>
      </div>
    </section>

    <section class="settings-section">
      <h3><span class="icon-document"></span> Viewers</h3>
      <div class="form-group">
        <label>Open PDF Method</label>
        <div class="radio-group">
          <label>
            <input
              type="radio"
              name="openMethod"
              value="system"
              v-model="settingsStore.settings.openMethod"
            />
            System Default App
          </label>
          <label>
            <input
              type="radio"
              name="openMethod"
              value="inner"
              v-model="settingsStore.settings.openMethod"
            />
            Built-in Viewer (Tabs)
          </label>
        </div>
      </div>
      <div class="form-group">
        <label>Open Guitar Pro Method</label>
        <div class="radio-group">
          <label>
            <input
              type="radio"
              name="openGpMethod"
              value="system"
              v-model="settingsStore.settings.openGpMethod"
            />
            System Default App
          </label>
          <label>
            <input
              type="radio"
              name="openGpMethod"
              value="inner"
              v-model="settingsStore.settings.openGpMethod"
            />
            Built-in Viewer (AlphaTab)
          </label>
        </div>
      </div>
    </section>

    <section class="settings-section" v-if="isAudioOutputSupported">
      <h3><span class="icon-volume"></span> Audio</h3>
      <div class="form-group">
        <label>Output Device</label>
        <select v-model="settingsStore.settings.audioDevice">
          <option value="default">Default</option>
          <option
            v-for="device in audioDevices"
            :key="device.deviceId"
            :value="device.deviceId"
          >
            {{ device.label || 'Unknown Device (' + device.deviceId.slice(0, 8) + '...)' }}
          </option>
        </select>
        <p class="hint">Applied to Guitar Pro playback</p>
      </div>
    </section>

    <section class="settings-section">
      <h3><span class="icon-keyboard"></span> Shortcuts</h3>
      <div class="form-group">
        <label>Key Bindings</label>
        <button class="btn" @click="uiStore.showKeyBindingsModal">Configure Key Bindings</button>
      </div>
    </section>

    <section class="settings-section">
      <h3><span class="icon-sync"></span> Auto Sync</h3>
      <div class="form-group">
        <label>
          <input type="checkbox" v-model="settingsStore.settings.autoSyncEnabled">
          Enable Auto Sync
        </label>
      </div>
      <div class="form-group" v-if="settingsStore.settings.autoSyncEnabled">
        <label>Sync Frequency</label>
        <select v-model="settingsStore.settings.autoSyncFrequency">
          <option value="startup">Every Startup</option>
          <option value="weekly">First Startup of Week</option>
          <option value="monthly">First Startup of Month</option>
          <option value="yearly">First Startup of Year</option>
        </select>
      </div>
      <div class="form-group">
        <label>Sync Strategy (for duplicates)</label>
        <select id="set-sync-strategy" v-model="settingsStore.settings.syncStrategy">
          <option value="skip">Skip (Keep existing)</option>
          <option value="overwrite">Overwrite (Prefer found files)</option>
        </select>
      </div>
      <div class="form-group">
        <label>Monitored Folders</label>
        <ul id="sync-path-list">
          <li v-for="(path, index) in settingsStore.settings.syncPaths" :key="index">
            <span>{{ path }}</span>
            <span class="delete-icon" @click="settingsStore.removeSyncPath(index)">
              <span class="icon-trash"></span>
            </span>
          </li>
        </ul>
        <button class="btn small" @click="handleAddSyncPath">+ Add Folder</button>
      </div>
      <div class="sync-actions">
        <button class="btn primary" @click="handleSync">Sync Now</button>
      </div>
    </section>

    <div class="settings-footer">
      <button class="btn primary" @click="handleSave">Save Changes</button>
    </div>
  </div>
</template>
