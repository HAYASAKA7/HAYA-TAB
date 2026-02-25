<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUIStore, useSettingsStore, useTabsStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const { t } = useI18n()
const uiStore = useUIStore()
const settingsStore = useSettingsStore()
const tabsStore = useTabsStore()
const { showToast } = useToast()

interface RemoteFile {
  name: string
  path: string
  size: number
  isDir: boolean
}

const files = ref<RemoteFile[]>([])
const loading = ref(false)
const selectedFiles = ref<Set<string>>(new Set())
const searchQuery = ref('')
const progress = ref<{ status: string, current?: number, total?: number, filename?: string, success?: number } | null>(null)

onMounted(() => {
  if (uiStore.cloudPickerModalVisible) {
    scanFiles()
  }
})

watch(() => uiStore.cloudPickerModalVisible, (visible) => {
  if (visible) {
    scanFiles()
    progress.value = null
    selectedFiles.value.clear()
    searchQuery.value = ''
    
    // Listen for progress events
    EventsOn('cloud-progress', (data: any) => {
      progress.value = data
      if (data.status === 'complete') {
        showToast(t('cloud.downloadComplete') + ` (${data.success} files)`)
        loading.value = false
        // Refresh local library
        tabsStore.refreshData()
        uiStore.hideCloudPickerModal()
      }
    })
  } else {
    EventsOff('cloud-progress')
  }
})

async function scanFiles() {
  if (!settingsStore.settings.webdavEnabled) {
    showToast(t('settings.enableWebdav'), 'error')
    uiStore.hideCloudPickerModal()
    return
  }

  loading.value = true
  files.value = []
  
  try {
    const result = await window.go.main.App.WebDAVScanRemoteFiles(
      settingsStore.settings.webdavUrl,
      settingsStore.settings.webdavUser,
      settingsStore.settings.webdavPassword,
      '/'
    )
    files.value = result || []
  } catch (err) {
    console.error(err)
    showToast(t('cloud.scanError') + ': ' + err, 'error')
  } finally {
    loading.value = false
  }
}

const filteredFiles = computed(() => {
  if (!searchQuery.value) return files.value
  const query = searchQuery.value.toLowerCase()
  return files.value.filter(f => f.name.toLowerCase().includes(query) || f.path.toLowerCase().includes(query))
})

function toggleSelection(path: string) {
  if (selectedFiles.value.has(path)) {
    selectedFiles.value.delete(path)
  } else {
    selectedFiles.value.add(path)
  }
}

function selectAll() {
  if (selectedFiles.value.size === filteredFiles.value.length) {
    selectedFiles.value.clear()
  } else {
    filteredFiles.value.forEach(f => selectedFiles.value.add(f.path))
  }
}

async function handleDownload() {
  if (selectedFiles.value.size === 0) return

  loading.value = true
  try {
    await window.go.main.App.WebDAVDownloadFiles(
      settingsStore.settings.webdavUrl,
      settingsStore.settings.webdavUser,
      settingsStore.settings.webdavPassword,
      Array.from(selectedFiles.value)
    )
    // Progress events will handle completion
  } catch (err) {
    loading.value = false
    showToast('Download failed: ' + err, 'error')
  }
}

function formatSize(bytes: number) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}
</script>

<template>
  <div v-if="uiStore.cloudPickerModalVisible" class="modal-overlay" @click.self="uiStore.hideCloudPickerModal">
    <div class="modal large">
      <h2>{{ t('cloud.title') }}</h2>
      
      <div class="modal-body">
        <div class="toolbar">
          <input 
            type="text" 
            v-model="searchQuery" 
            :placeholder="t('search.placeholder')"
            class="search-input"
          />
          <button class="btn" @click="scanFiles" :disabled="loading">
            <span class="icon-sync" :class="{ spinning: loading }"></span>
          </button>
        </div>

        <div class="file-list-container">
          <div v-if="loading && !progress" class="loading-state">
            <span class="spinner"></span> {{ t('cloud.scanning') }}
          </div>
          <div v-else-if="progress" class="progress-state">
            <div class="progress-bar">
              <div 
                class="progress-fill" 
                :style="{ width: (progress.total ? (progress.current || 0) / progress.total * 100 : 0) + '%' }"
              ></div>
            </div>
            <p>{{ t('cloud.downloading') }} {{ progress.current }} / {{ progress.total }}</p>
            <p v-if="progress.filename">{{ progress.filename }}</p>
          </div>
          <table v-else class="file-table">
            <thead>
              <tr>
                <th width="40"><input type="checkbox" @change="selectAll" :checked="selectedFiles.size > 0 && selectedFiles.size === filteredFiles.length"></th>
                <th>{{ t('cloud.name') }}</th>
                <th>{{ t('cloud.path') }}</th>
                <th width="100">{{ t('cloud.size') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="file in filteredFiles" :key="file.path" :class="{ selected: selectedFiles.has(file.path) }" @click="toggleSelection(file.path)">
                <td><input type="checkbox" :checked="selectedFiles.has(file.path)" @click.stop="toggleSelection(file.path)"></td>
                <td>{{ file.name }}</td>
                <td class="path-col">{{ file.path }}</td>
                <td>{{ formatSize(file.size) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="modal-actions">
        <button class="btn" @click="uiStore.hideCloudPickerModal" :disabled="loading">{{ t('confirm.cancel') }}</button>
        <button class="btn primary" @click="handleDownload" :disabled="selectedFiles.size === 0 || loading">
          {{ t('cloud.downloadSelected') }} ({{ selectedFiles.size }})
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal.large {
  width: 800px;
  max-width: 90vw;
  height: 80vh; /* Fixed height for browser */
  display: flex;
  flex-direction: column;
}

.modal-body {
  margin-top: 16px;
  flex: 1; /* Take remaining space */
  overflow: hidden; /* Prevent modal body scroll, handle inside components */
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar {
  display: flex;
  gap: 1rem;
}

.search-input {
  flex: 1;
  padding: 8px;
  background-color: var(--bg);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text);
}

.file-list-container {
  flex: 1;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 4px;
  position: relative;
  background-color: var(--bg);
}

.file-table {
  width: 100%;
  border-collapse: collapse;
}

.file-table th, .file-table td {
  padding: 0.75rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
  color: var(--text);
}

.file-table th {
  background: var(--card-bg);
  position: sticky;
  top: 0;
  font-weight: 600;
}

.file-table tr:hover {
  background: var(--hover);
  cursor: pointer;
}

.file-table tr.selected {
  background: rgba(var(--primary-rgb), 0.1); /* Assuming variable exists or fallback */
  background: var(--hover); /* Fallback */
}

.path-col {
  color: var(--text-muted);
  font-size: 0.9em;
}

.loading-state, .progress-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
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

.spinning {
  animation: spin 1s linear infinite;
  display: inline-block;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.progress-bar {
  width: 300px;
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
</style>
