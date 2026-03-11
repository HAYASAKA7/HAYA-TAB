<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUIStore, useSettingsStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { CloudService } from '@/services'

const { t } = useI18n()
const uiStore = useUIStore()
const settingsStore = useSettingsStore()
const { showToast } = useToast()

const url = ref('')
const user = ref('')
const password = ref('')
const testing = ref(false)

onMounted(() => {
  if (uiStore.webdavModalVisible) {
    url.value = settingsStore.settings.webdavUrl
    user.value = settingsStore.settings.webdavUser
    password.value = settingsStore.settings.webdavPassword
  }
})

watch(() => uiStore.webdavModalVisible, (visible) => {
  if (visible) {
    url.value = settingsStore.settings.webdavUrl
    user.value = settingsStore.settings.webdavUser
    password.value = settingsStore.settings.webdavPassword
  }
})

async function testConnection() {
  if (!url.value) {
    showToast(t('settings.pleaseEnterUrl', 'Please enter WebDAV URL'), 'warning')
    return
  }

  testing.value = true
  try {
    await CloudService.testConnection(url.value, user.value, password.value)
    showToast(t('settings.connectionSuccess'), 'success')
  } catch (err) {
    // Extract error message
    const errorMsg = String(err)
    showToast(t('settings.connectionFailed') + ': ' + errorMsg, 'error')
  } finally {
    testing.value = false
  }
}

async function save() {
  const oldUrl = settingsStore.settings.webdavUrl
  const oldUser = settingsStore.settings.webdavUser
  const oldPass = settingsStore.settings.webdavPassword

  settingsStore.settings.webdavUrl = url.value
  settingsStore.settings.webdavUser = user.value
  settingsStore.settings.webdavPassword = password.value

  if (oldUrl !== url.value || oldUser !== user.value || oldPass !== password.value) {
    try {
      await settingsStore.saveSettings()
      showToast(t('settings.settingsSaved'))

      // If WebDAV is enabled, immediately check connection status
      if (settingsStore.settings.webdavEnabled) {
        await settingsStore.checkWebDAVStatus()
      }
    } catch (err) {
      showToast(t('settings.errorSaving') + ': ' + err, 'error')
    }
  }

  uiStore.hideWebdavModal()
}

function cancel() {
  uiStore.hideWebdavModal()
}
</script>

<template>
  <div v-if="uiStore.webdavModalVisible" class="modal-overlay" @click.self="cancel">
    <div class="modal">
      <h2>{{ t('settings.configureWebdav') }}</h2>
      
      <div class="modal-body">
        <div class="form-group">
          <label>URL</label>
          <input type="text" v-model="url" placeholder="https://dav.example.com/remote.php/webdav/">
        </div>
        <div class="form-group">
          <label>{{ t('settings.username') }}</label>
          <input type="text" v-model="user">
        </div>
        <div class="form-group">
          <label>{{ t('settings.password') }}</label>
          <input type="password" v-model="password">
        </div>
      </div>

      <div class="modal-actions">
        <button class="btn" @click="testConnection" :disabled="testing" style="margin-right: auto;">
          <span v-if="testing" class="spinner-sm"></span>
          {{ t('settings.testConnection') }}
        </button>
        <button class="btn" @click="cancel">{{ t('confirm.cancel') }}</button>
        <button class="btn primary" @click="save">{{ t('confirm.save') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal {
  width: auto;
  min-width: 400px;
  max-width: min(90vw, 500px);
}

.modal-body {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-weight: 500;
  color: var(--text);
  font-size: 0.9rem;
}

.form-group input {
  padding: 10px;
  background-color: var(--bg);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text);
  font-size: 0.95rem;
}

.form-group input:focus {
  outline: none;
  border-color: var(--primary);
}

.spinner-sm {
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  display: inline-block;
  margin-right: 0.5rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
