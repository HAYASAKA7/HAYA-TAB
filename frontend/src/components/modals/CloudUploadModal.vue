<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUIStore, useSettingsStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { CloudService } from '@/services'

const { t } = useI18n()
const uiStore = useUIStore()
const settingsStore = useSettingsStore()
const { showToast, showErrorToast } = useToast()

const directories = ref<string[]>([])
const loading = ref(false)
const selectedDir = ref('/')
const progress = ref<{ status: string, current?: number, total?: number, filename?: string, success?: number } | null>(null)

onMounted(() => {
  if (uiStore.cloudUploadModalVisible) {
    listDirectories('/')
  }
})

watch(() => uiStore.cloudUploadModalVisible, (visible) => {
  if (visible) {
    listDirectories('/')
    progress.value = null
    selectedDir.value = '/'
    
    EventsOn('cloud-upload-progress', (data: any) => {
      progress.value = data
      if (data.status === 'complete') {
        showToast(t('cloud.uploadComplete') + ` (${data.success} files)`)
        loading.value = false
        uiStore.hideCloudUploadModal()
      }
    })
  } else {
    EventsOff('cloud-upload-progress')
  }
})

async function listDirectories(path: string) {
  if (!settingsStore.settings.webdavEnabled) return

  loading.value = true
  try {
    const result = await CloudService.listRemoteDirectories(
      settingsStore.settings.webdavUrl,
      settingsStore.settings.webdavUser,
      settingsStore.settings.webdavPassword,
      path
    )
    directories.value = result || []
    // Add parent directory option if not root
    if (path !== '/' && path !== '') {
      directories.value.unshift('..')
    }
  } catch (err) {
    console.error(err)
    showErrorToast(err, t('cloud.listDirectoriesFailed'))
  } finally {
    loading.value = false
  }
}

async function handleDirClick(dir: string) {
  if (dir === '..') {
    // Go up one level (simple string manipulation for now)
    const parts = selectedDir.value.split('/').filter(p => p)
    parts.pop()
    selectedDir.value = parts.length > 0 ? '/' + parts.join('/') : '/'
  } else {
    selectedDir.value = dir
  }
  await listDirectories(selectedDir.value)
}

async function handleUpload() {
  if (uiStore.cloudUploadFiles.length === 0) return

  loading.value = true
  try {
    await CloudService.uploadFiles(
      settingsStore.settings.webdavUrl,
      settingsStore.settings.webdavUser,
      settingsStore.settings.webdavPassword,
      uiStore.cloudUploadFiles,
      selectedDir.value
    )
  } catch (err) {
    loading.value = false
    showErrorToast(err, t('cloud.uploadFailed'))
  }
}
</script>

<template>
  <div v-if="uiStore.cloudUploadModalVisible" class="modal-overlay" @click.self="uiStore.hideCloudUploadModal">
    <div class="modal">
      <h2>{{ t('cloud.uploadTitle') }}</h2>
      
      <div class="modal-body">
        <p class="instruction">{{ t('cloud.selectDestination') }}:</p>
        <div class="current-path">{{ selectedDir }}</div>

        <div v-if="loading && !progress" class="loading-state">
          <span class="spinner"></span>
        </div>
        <div v-else-if="progress" class="progress-state">
          <div class="progress-bar">
            <div 
              class="progress-fill" 
              :style="{ width: (progress.total ? (progress.current || 0) / progress.total * 100 : 0) + '%' }"
            ></div>
          </div>
          <p>{{ t('cloud.uploading') }} {{ progress.current }} / {{ progress.total }}</p>
          <p v-if="progress.filename">{{ progress.filename }}</p>
        </div>
        <div v-else class="dir-list">
          <div 
            v-for="dir in directories" 
            :key="dir" 
            class="dir-item"
            @click="handleDirClick(dir)"
          >
            <span class="icon-folder"></span>
            <span class="dir-name">{{ dir === '..' ? '.. (Up)' : dir.split('/').pop() }}</span>
          </div>
          <div v-if="directories.length === 0" class="empty-state">
            No subdirectories found.
          </div>
        </div>
      </div>

      <div class="modal-actions">
        <button class="btn" @click="uiStore.hideCloudUploadModal" :disabled="loading">{{ t('confirm.cancel') }}</button>
        <button class="btn primary" @click="handleUpload" :disabled="loading">
          {{ t('cloud.upload') }} ({{ uiStore.cloudUploadFiles.length }})
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal {
  width: 500px;
  max-width: 90vw;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.modal-body {
  margin-top: 16px;
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.instruction {
  color: var(--text);
  font-weight: 500;
}

.current-path {
  background: var(--bg);
  padding: 8px;
  border: 1px solid var(--border);
  border-radius: 4px;
  font-family: monospace;
  word-break: break-all;
  color: var(--text);
}

.dir-list {
  flex: 1;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 4px;
  background-color: var(--bg);
}

.dir-item {
  padding: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  border-bottom: 1px solid var(--border);
  color: var(--text);
}

.dir-item:hover {
  background: var(--hover);
}

.dir-item:last-child {
  border-bottom: none;
}

.icon-folder {
  color: var(--primary);
  font-size: 1.2em;
}

.loading-state, .progress-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  gap: 1rem;
  color: var(--text);
}

.spinner {
  width: 24px;
  height: 24px;
  border: 3px solid var(--border);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.progress-bar {
  width: 80%;
  height: 8px;
  background: var(--border);
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--primary);
  transition: width 0.3s ease;
}

.empty-state {
  padding: 2rem;
  text-align: center;
  color: var(--text-muted);
}
</style>
