<script setup lang="ts">
import { Events } from "@wailsio/runtime"
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useSettingsStore, useUIStore, useViewersStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { UpdateService } from '@/services'
import '@/services/MidiService'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import HomeView from '@/views/HomeView.vue'
import LibraryView from '@/views/LibraryView.vue'
import SettingsView from '@/components/SettingsView.vue'
import PluginsView from '@/components/PluginsView.vue'
import PdfViewer from '@/components/viewers/PdfViewer.vue'
import GpViewer from '@/components/viewers/GpViewer.vue'
import Toast from '@/components/common/Toast.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import EditTabModal from '@/components/modals/EditTabModal.vue'
import CategoryModal from '@/components/modals/CategoryModal.vue'
import MoveTabModal from '@/components/modals/MoveTabModal.vue'
import BatchMoveModal from '@/components/modals/BatchMoveModal.vue'
import ConfirmModal from '@/components/modals/ConfirmModal.vue'
import KeyBindingModal from '@/components/modals/KeyBindingModal.vue'
import CloudPickerModal from '@/components/modals/CloudPickerModal.vue'
import CloudUploadModal from '@/components/modals/CloudUploadModal.vue'
import WebDAVModal from '@/components/modals/WebDAVModal.vue'
import PluginSettingsModal from '@/components/modals/PluginSettingsModal.vue'
import BatchActionBar from '@/components/BatchActionBar.vue'
import SyncTaskToast from '@/components/common/SyncTaskToast.vue'
import { PluginService } from '@/services/PluginService'

const tabsStore = useTabsStore()
const settingsStore = useSettingsStore()
const uiStore = useUIStore()
const viewersStore = useViewersStore()
const { showToast, showErrorToast } = useToast()
const { t } = useI18n()

onMounted(async () => {
  await tabsStore.refreshData()
  await settingsStore.loadSettings()

  // Check if plugins exist
  try {
    uiStore.hasPlugins = await PluginService.hasPlugins()
  } catch (e) {
    console.error('Failed to check plugins:', e)
  }

  // Start WebDAV status monitoring if enabled
  if (settingsStore.settings.webdavEnabled) {
    settingsStore.startWebDAVStatusCheck()
  }

  // Check for updates
  const updateInfo = await UpdateService.checkForUpdates()
  if (updateInfo) {
    uiStore.updateInfo = updateInfo
  }

  // Event listeners
  Events.On('tab-updated', (ev: any) => {
    const hasDataField = ev && typeof ev === 'object' && Object.prototype.hasOwnProperty.call(ev, 'data')
    const payload = hasDataField ? ev.data : ev

    // Skip full refresh when event has no payload (used by WebDAV volume initialization).
    // This preserves scroll position and avoids jumping back to top.
    if (payload === null || payload === undefined) {
      return
    }

    // Support both single-item and array payloads from backend emitters.
    const items = Array.isArray(payload) ? payload : [payload]
    const tabs = items.filter((item: any) => item && item.id)

    if (tabs.length === 0) {
      return
    }

    for (const tab of tabs) {
      const exists = tabsStore.tabs.some(t => t.id === tab.id)
      if (exists) {
        tabsStore.updateTabInPlace(tab.id, tab)
      } else {
        tabsStore.addTabsInPlace([tab])
      }
    }
  })

  Events.On('cover-error', (ev: any) => {
    const msg = ev.data ? (Array.isArray(ev.data) ? ev.data[0] : ev.data) : ''
    showToast(msg, 'error')
  })

  Events.On('sync-complete', (ev: any) => {
    const msg = ev.data ? (Array.isArray(ev.data) ? ev.data[0] : ev.data) : ''
    showToast(msg, 'info')
    tabsStore.refreshData()
  })

  Events.On('file-changes-detected', (ev: any) => {
    const msg = ev.data ? (Array.isArray(ev.data) ? ev.data[0] : ev.data) : ''
    showToast(msg + ' - ' + t('toast.clickSyncToUpdate'), 'info')
  })

  Events.On('migration-completed', (ev: any) => {
    const target = ev.data ? (Array.isArray(ev.data) ? ev.data[0] : ev.data) : ''
    tabsStore.refreshData()
    // For covers migration, we might need to clear browser cache, but simple reload is enough
    if (target === 'covers') {
      window.location.reload()
    }
  })

  // Listen for cloud download completion (handle toast globally to avoid duplicates)
  Events.On('cloud-download-single', (ev: any) => {
    const data = ev.data ? (Array.isArray(ev.data) ? ev.data[0] : ev.data) : ev
    if (data.status === 'complete') {
      showToast(t('cloud.downloadSuccess'), 'success')
      tabsStore.refreshData()
    } else if (data.status === 'error') {
      showErrorToast({ i18nKey: data.messageKey, i18nArgs: data.errorArgs }, t('cloud.downloadFailed'))
    }
  })

  // Tab deletion events are handled in-place by the store to preserve scroll position
  // No need to listen for 'tab-deleted' or 'tabs-deleted' events here
})

function isViewActive(viewType: string): boolean {
  if (viewType === 'home') {
    return uiStore.currentView === 'home'
  }
  if (viewType === 'library') {
    return uiStore.currentView === 'library'
  }
  if (viewType === 'settings') {
    return uiStore.currentView === 'settings'
  }
  if (viewType === 'plugins') {
    return uiStore.currentView === 'plugins'
  }
  if (viewType === 'pdf') {
    return uiStore.currentView.startsWith('pdf-')
  }
  if (viewType === 'gp') {
    return uiStore.currentView.startsWith('gp-')
  }
  return false
}
</script>

<template>
  <div id="app-layout" :class="{ 'sidebar-collapsed': uiStore.sidebarCollapsed }">
    <AppSidebar />

    <main id="main-content">
      <!-- Home View -->
      <div
        id="view-home"
        class="view"
        :class="{ hidden: !isViewActive('home') }"
      >
        <HomeView />
      </div>

      <!-- Library View -->
      <div
        id="view-library"
        class="view"
        :class="{ hidden: !isViewActive('library') }"
      >
        <LibraryView />
      </div>

      <!-- Settings View -->
      <div
        id="view-settings"
        class="view"
        :class="{ hidden: !isViewActive('settings') }"
      >
        <SettingsView />
      </div>

      <!-- Plugins View -->
      <div
        id="view-plugins"
        class="view"
        :class="{ hidden: !isViewActive('plugins') }"
      >
        <PluginsView />
      </div>

      <!-- PDF Views Container -->
      <div
        id="pdf-views-container"
        :class="{ active: isViewActive('pdf') }"
      >
        <PdfViewer
          v-for="tabId in viewersStore.openedTabs"
          :key="`pdf-${tabId}`"
          :tab-id="tabId"
          :visible="uiStore.currentView === `pdf-${tabId}`"
        />
      </div>

      <!-- GP Views Container -->
      <div
        id="gp-views-container"
        :class="{ active: isViewActive('gp') }"
      >
        <GpViewer
          v-for="tabId in viewersStore.openedTabs"
          :key="`gp-${tabId}`"
          :tab-id="tabId"
          :visible="uiStore.currentView === `gp-${tabId}`"
        />
      </div>
    </main>

    <!-- Batch Action Bar -->
    <BatchActionBar />

    <!-- Modals -->
    <EditTabModal />
    <CategoryModal />
    <MoveTabModal />
    <BatchMoveModal />
    <ConfirmModal />
    <KeyBindingModal />
    <CloudPickerModal />
    <CloudUploadModal />
    <WebDAVModal />
    <PluginSettingsModal />

    <!-- Toast & Context Menu -->
    <Toast />
    <SyncTaskToast />
    <ContextMenu />
  </div>
</template>

<style>
@import '@/assets/style.css';
@import '@/assets/icons.css';
</style>
