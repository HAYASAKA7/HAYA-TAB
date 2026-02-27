<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Tab, ContextMenuItem } from '@/types'
import { useTabsStore, useUIStore, useViewersStore, useSettingsStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'
import { useToast } from '@/composables/useToast'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps<{
  tab: Tab
}>()

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const viewersStore = useViewersStore()
const settingsStore = useSettingsStore()
const contextMenu = useContextMenu()
const { showToast } = useToast()

const fileServerPort = ref(0)
const coverError = ref(false)
const coverTimestamp = ref(Date.now()) // Used for cache busting when cover changes
const coverUrl = computed(() => {
  if (!props.tab.coverPath || coverError.value || !fileServerPort.value) return ''
  // Use the file server endpoint for cover images
  return `http://127.0.0.1:${fileServerPort.value}/api/cover/${props.tab.id}?t=${coverTimestamp.value}`
})
const isSelected = computed(() => tabsStore.isTabSelected(props.tab.id))

// Compute display label for file type badge
const fileTypeLabel = computed(() => {
  const ext = props.tab.filePath.toLowerCase().split('.').pop() || ''
  if (props.tab.type === 'pdf') return 'PDF'
  if (ext === 'xml' || ext === 'musicxml') return 'XML'
  if (ext === 'mxl') return 'MXL'
  return 'GP'
})

// Check if this is a cloud tab that's offline
const isCloudOffline = computed(() =>
  props.tab.isCloud && !settingsStore.webdavConnected
)

// Reset error state and update timestamp when cover path changes
watch(() => props.tab.coverPath, () => {
  coverError.value = false
  coverTimestamp.value = Date.now()
})

onMounted(async () => {
  try {
    fileServerPort.value = await window.go.app.App.GetFileServerPort()
  } catch (e) {
    console.error('Failed to get file server port:', e)
  }

  // Listen for cloud download completion events (only update data, toast is handled globally in App.vue)
  EventsOn('cloud-download-single', (data: any) => {
    if (data.status === 'complete' && data.tabId === props.tab.id && data.tab) {
      // In-place update: update the tab properties without full refresh
      tabsStore.updateTabInPlace(data.tabId, {
        isCloud: data.tab.isCloud,
        isManaged: data.tab.isManaged,
        filePath: data.tab.filePath,
        categoryIds: data.tab.categoryIds
      })
    }
  })
})

onUnmounted(() => {
  EventsOff('cloud-download-single')
})

function handleClick(e: MouseEvent) {
  // Block opening cloud tabs when offline
  if (isCloudOffline.value) {
    e.preventDefault()
    e.stopPropagation()
    showToast(t('cloud.offlineCannotOpen'), 'warning')
    return
  }

  if (tabsStore.isBatchSelectMode) {
    tabsStore.toggleTabSelection(props.tab.id)
  } else {
    openTab()
  }
}

async function openTab() {
  const settings = settingsStore.settings

  if (settings.openMethod === 'inner' && props.tab.type === 'pdf') {
    openInternalTab()
  } else if (settings.openGpMethod === 'inner' && props.tab.type === 'gp') {
    openInternalTab()
  } else {
    try {
      await window.go.app.App.OpenTab(props.tab.id)
    } catch (err) {
      console.error(err)
      showToast(t('contextMenu.failedToOpen'), 'error')
    }
  }
}

async function openInternalTab() {
  try {
    // Notify backend to update timestamp
    await (window.go.app.App as any).MarkAsOpened(props.tab.id)
  } catch (err) {
    console.warn('Failed to mark tab as opened:', err)
  }

  viewersStore.openTab(props.tab)
  const prefix = props.tab.type === 'pdf' ? 'pdf' : 'gp'
  uiStore.switchView(`${prefix}-${props.tab.id}`)
}

function handleContextMenu(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()

  if (tabsStore.isBatchSelectMode) return

  const items: ContextMenuItem[] = []

  // Cloud tab specific options
  if (props.tab.isCloud) {
    // Download to local option (always available for cloud tabs)
    items.push({
      label: t('cloud.downloadToLocal'),
      action: () => downloadToLocal()
    })

    // Only show open options if online
    if (!isCloudOffline.value) {
      items.push(
        { label: t('contextMenu.openWithInner'), action: () => openInternalTab() }
      )
    }

    items.push(
      { label: t('contextMenu.editMetadata'), action: () => uiStore.showEditModal(props.tab) },
      { type: 'separator' },
      { label: t('contextMenu.removeTab'), action: () => confirmDelete() }
    )
  } else {
    // Regular local tab options
    items.push(
      { label: t('contextMenu.openWithSystem'), action: () => window.go.app.App.OpenTab(props.tab.id) },
      { label: t('contextMenu.openWithInner'), action: () => openInternalTab() },
      { label: t('contextMenu.editMetadata'), action: () => uiStore.showEditModal(props.tab) },
      { label: t('contextMenu.addToCategory'), action: () => uiStore.showMoveModal(props.tab.id) }
    )

    if (tabsStore.currentCategoryId) {
      items.push({
        label: t('contextMenu.removeFromCategory'),
        action: async () => {
          await tabsStore.removeTabFromCategory(props.tab.id, tabsStore.currentCategoryId)
          showToast(t('contextMenu.removedFromCategory'))
        }
      })
    }

    items.push(
      { label: t('contextMenu.exportTab'), action: () => exportTab() },
      { label: t('cloud.uploadTitle'), action: () => uiStore.showCloudUploadModal([props.tab.filePath]) },
      { type: 'separator' },
      { label: props.tab.isManaged ? t('contextMenu.deleteTab') : t('contextMenu.unlinkTab'), action: () => confirmDelete() }
    )
  }

  contextMenu.show(e.pageX, e.pageY, items)
}

async function downloadToLocal() {
  try {
    showToast(t('cloud.downloadingToLocal'), 'info')
    await window.go.app.App.DownloadCloudTabToLocal(props.tab.id)
    // Success/error handling is done via cloud-download-single event listener
  } catch (err) {
    console.error('Failed to download cloud tab:', err)
    // Error toast is handled by event listener, only log here
  }
}

async function exportTab() {
  const dest = await window.go.app.App.SelectFolder()
  if (dest) {
    await window.go.app.App.ExportTab(props.tab.id, dest)
    showToast(t('contextMenu.exported'))
  }
}

function confirmDelete() {
  const title = props.tab.isManaged ? t('contextMenu.deleteTab') : t('contextMenu.unlinkTab')
  const message = props.tab.isManaged
    ? `${t('contextMenu.confirmDeleteTab', { title: props.tab.title })}<br><br><span class="warning-text">${t('contextMenu.confirmDeleteTabWarning')}</span>`
    : `${t('contextMenu.confirmUnlinkTab', { title: props.tab.title })}<br><br>${t('contextMenu.confirmUnlinkTabInfo')}`
  const btnText = props.tab.isManaged ? t('confirm.delete') : t('contextMenu.unlinkTab')

  uiStore.showConfirmModal(title, message, btnText, true, async () => {
    await tabsStore.deleteTabInPlace(props.tab.id)
  })
}

function handleCheckboxClick(e: Event) {
  e.stopPropagation()
  tabsStore.toggleTabSelection(props.tab.id)
}
</script>

<template>
  <div
    class="tab-card"
    :class="{
      selected: tabsStore.isBatchSelectMode && isSelected,
      'cloud-tab': tab.isCloud,
      'cloud-offline': isCloudOffline
    }"
    @click="handleClick"
    @contextmenu="handleContextMenu"
  >
    <!-- Offline overlay for cloud tabs -->
    <div v-if="isCloudOffline" class="offline-overlay">
      <div class="offline-icon-wrapper">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="cloud-offline-icon">
          <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM19 18H6c-2.21 0-4-1.79-4-4 0-2.05 1.53-3.76 3.56-3.97l1.07-.11.5-.95C8.08 7.14 9.94 6 12 6c2.62 0 4.88 1.86 5.39 4.43l.3 1.5 1.53.11c1.56.1 2.78 1.41 2.78 2.96 0 1.65-1.35 3-3 3z"/>
        </svg>
        <svg class="offline-slash" viewBox="0 0 24 24" fill="none" stroke="var(--danger)" stroke-width="2.5">
          <line x1="4" y1="4" x2="20" y2="20"/>
        </svg>
      </div>
    </div>

    <!-- Cloud indicator badge -->
    <div v-if="tab.isCloud && !isCloudOffline" class="cloud-badge">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="cloud-icon-small">
        <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM19 18H6c-2.21 0-4-1.79-4-4 0-2.05 1.53-3.76 3.56-3.97l1.07-.11.5-.95C8.08 7.14 9.94 6 12 6c2.62 0 4.88 1.86 5.39 4.43l.3 1.5 1.53.11c1.56.1 2.78 1.41 2.78 2.96 0 1.65-1.35 3-3 3z"/>
      </svg>
    </div>

    <!-- Checkbox for batch mode -->
    <div
      v-if="tabsStore.isBatchSelectMode"
      class="select-checkbox"
      :class="{ checked: isSelected }"
      @click="handleCheckboxClick"
    >
      <span class="icon-checkbox"></span>
    </div>


    <!-- Cover -->
    <div class="cover-wrapper">
      <div class="placeholder-cover">
        <img
          v-if="coverUrl"
          :src="coverUrl"
          class="cover-img"
          loading="lazy"
          @error="coverError = true"
        />
        <span v-else class="icon-music icon-xl"></span>
      </div>
    </div>

    <!-- Info -->
    <div class="info">
      <div class="title" :title="tab.title">{{ tab.title }}</div>
      <div class="artist" :title="tab.artist">{{ tab.artist }}</div>
      <div class="type-badge">{{ fileTypeLabel }}</div>
      <div v-if="tab.tag" class="tag-badge" :title="tab.tag">{{ tab.tag }}</div>
    </div>
  </div>
</template>

<style scoped>
/* Cloud tab styles */
.cloud-tab {
  position: relative;
}

.cloud-badge {
  position: absolute;
  top: 8px;
  left: 8px;
  z-index: 10;
  background: var(--primary);
  border-radius: 50%;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cloud-icon-small {
  width: 14px;
  height: 14px;
  color: white;
}

/* Offline state */
.cloud-offline {
  position: relative;
}

.cloud-offline .cover-wrapper,
.cloud-offline .info {
  opacity: 0.5;
  filter: grayscale(50%);
}

.offline-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 15;
  border-radius: var(--radius);
  pointer-events: none;
}

.offline-icon-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cloud-offline-icon {
  width: 48px;
  height: 48px;
  color: var(--text-muted);
}

.offline-slash {
  position: absolute;
  width: 48px;
  height: 48px;
}

/* Ensure context menu still works on offline cards */
.cloud-offline {
  cursor: not-allowed;
}
</style>
