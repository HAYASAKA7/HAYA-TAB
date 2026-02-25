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

const currentPath = ref('/')
const files = ref<RemoteFile[]>([])
const loading = ref(false)
const selectedFiles = ref<Set<string>>(new Set())
const searchQuery = ref('')
const progress = ref<{ status: string, current?: number, total?: number, filename?: string, success?: number, skipped?: number, errors?: number } | null>(null)

// Breadcrumbs computation
const breadcrumbs = computed(() => {
  const parts = currentPath.value.split('/').filter(p => p)
  const crumbs = [{ name: 'Root', path: '/' }]
  let current = ''
  for (const part of parts) {
    current += '/' + part
    crumbs.push({ name: part, path: current })
  }
  return crumbs
})

onMounted(() => {
  if (uiStore.cloudPickerModalVisible) {
    loadDirectory('/')
  }
})

watch(() => uiStore.cloudPickerModalVisible, (visible) => {
  if (visible) {
    currentPath.value = '/'
    loadDirectory('/')
    progress.value = null
    selectedFiles.value.clear()
    searchQuery.value = ''
    
    // Listen for progress events
    EventsOn('cloud-progress', (data: any) => {
      progress.value = data
      if (data.status === 'complete') {
        let msg = t('cloud.downloadComplete') + ` (${data.success} success`
        if (data.skipped) msg += `, ${data.skipped} skipped`
        if (data.errors) msg += `, ${data.errors} errors`
        msg += ')'
        
        showToast(msg, data.errors > 0 ? 'warning' : 'success')
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

async function loadDirectory(path: string) {
  if (!settingsStore.settings.webdavEnabled) {
    showToast(t('settings.enableWebdav'), 'error')
    uiStore.hideCloudPickerModal()
    return
  }

  loading.value = true
  files.value = []
  currentPath.value = path
  
  try {
    // Use the new non-recursive list method
    const result = await window.go.main.App.WebDAVListDir(
      settingsStore.settings.webdavUrl,
      settingsStore.settings.webdavUser,
      settingsStore.settings.webdavPassword,
      path
    )
    
    // Sort: Folders first, then files
    files.value = (result || []).sort((a: RemoteFile, b: RemoteFile) => {
      if (a.isDir === b.isDir) return a.name.localeCompare(b.name)
      return a.isDir ? -1 : 1
    })
  } catch (err) {
    console.error(err)
    showToast(t('cloud.scanError') + ': ' + err, 'error')
    // Go up if failed (e.g. permission error on folder)
    if (path !== '/') {
      goUp()
    }
  } finally {
    loading.value = false
  }
}

function goUp() {
  const parts = currentPath.value.split('/').filter(p => p)
  if (parts.length > 0) {
    parts.pop()
    loadDirectory('/' + parts.join('/'))
  } else {
    loadDirectory('/')
  }
}

const filteredFiles = computed(() => {
  if (!searchQuery.value) return files.value
  const query = searchQuery.value.toLowerCase()
  return files.value.filter(f => f.name.toLowerCase().includes(query))
})

const currentFilesOnly = computed(() => {
  return filteredFiles.value.filter(f => !f.isDir)
})

function navigateTo(path: string) {
  loadDirectory(path)
}

function toggleSelection(path: string) {
  if (selectedFiles.value.has(path)) {
    selectedFiles.value.delete(path)
  } else {
    selectedFiles.value.add(path)
  }
}

function selectAll() {
  const filePaths = currentFilesOnly.value.map(f => f.path)
  const allSelected = filePaths.every(p => selectedFiles.value.has(p))
  
  if (allSelected) {
    filePaths.forEach(p => selectedFiles.value.delete(p))
  } else {
    filePaths.forEach(p => selectedFiles.value.add(p))
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
           <!-- Breadcrumbs -->
          <div class="breadcrumbs">
             <span 
              v-for="(crumb, index) in breadcrumbs" 
              :key="crumb.path" 
              class="crumb"
              :class="{ active: index === breadcrumbs.length - 1 }"
              @click="loadDirectory(crumb.path)"
            >
              {{ crumb.name }}
            </span>
          </div>
          
          <div class="actions">
             <input 
              type="text" 
              v-model="searchQuery" 
              :placeholder="t('search.placeholder')"
              class="search-input"
            />
            <button class="btn icon-btn" @click="loadDirectory(currentPath)" :disabled="loading">
              <span class="icon-sync" :class="{ spinning: loading }"></span>
            </button>
          </div>
        </div>

        <div class="file-list-container">
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
            <p>{{ t('cloud.downloading') }} {{ progress.current }} / {{ progress.total }}</p>
            <p v-if="progress.filename">{{ progress.filename }}</p>
          </div>
          <table v-else class="file-table">
            <thead>
              <tr>
                <th width="40"><input type="checkbox" @change="selectAll" :checked="currentFilesOnly.length > 0 && currentFilesOnly.every(f => selectedFiles.has(f.path))" :disabled="currentFilesOnly.length === 0"></th>
                <th>{{ t('cloud.name') }}</th>
                <th width="100">{{ t('cloud.size') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="currentPath !== '/'" @click="goUp" class="folder-row">
                <td></td>
                <td colspan="2">..</td>
              </tr>
              <tr 
                v-for="file in filteredFiles" 
                :key="file.path" 
                :class="{ selected: selectedFiles.has(file.path), folder: file.isDir }" 
                @click="file.isDir ? navigateTo(file.path) : toggleSelection(file.path)"
              >
                <td @click.stop>
                  <input 
                    v-if="!file.isDir" 
                    type="checkbox" 
                    :checked="selectedFiles.has(file.path)" 
                    @change="toggleSelection(file.path)"
                  >
                  <span v-else class="icon-folder"></span>
                </td>
                <td>{{ file.name }}</td>
                <td>{{ file.isDir ? '-' : formatSize(file.size) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="modal-actions">
        <div class="selected-count" v-if="selectedFiles.size > 0">
          {{ selectedFiles.size }} {{ t('cloud.filesSelected') }}
        </div>
        <div style="flex: 1"></div>
        <button class="btn" @click="uiStore.hideCloudPickerModal">{{ t('confirm.cancel') }}</button>
        <button class="btn primary" @click="handleDownload" :disabled="selectedFiles.size === 0 || loading">
          {{ t('cloud.downloadSelected') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal.large {
  width: 800px;
  max-width: 90vw;
  height: 80vh;
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

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  background: var(--bg-alt);
  padding: 8px;
  border-radius: 4px;
}

.breadcrumbs {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 0.9em;
}

.crumb {
  cursor: pointer;
  color: var(--primary);
  padding: 2px 4px;
  border-radius: 4px;
}

.crumb:hover {
  background: var(--hover);
  text-decoration: underline;
}

.crumb.active {
  color: var(--text);
  font-weight: 600;
  cursor: default;
  text-decoration: none;
}

.crumb:not(:last-child)::after {
  content: '/';
  margin-left: 4px;
  color: var(--text-muted);
}

.actions {
  display: flex;
  gap: 8px;
}

.search-input {
  width: 200px;
  padding: 6px;
  background-color: var(--bg);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text);
}

.icon-btn {
  padding: 6px 10px;
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
  z-index: 1;
}

.file-table tr:hover {
  background: var(--hover);
  cursor: pointer;
}

.file-table tr.selected {
  background: rgba(var(--primary-rgb), 0.1); 
  background: var(--hover);
}

.folder-row {
  background-color: rgba(0, 0, 0, 0.02);
}

.folder {
  font-weight: 500;
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

.selected-count {
  font-size: 0.9em;
  color: var(--text-muted);
}
</style>
