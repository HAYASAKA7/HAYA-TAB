<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useSettingsStore, useUIStore, useViewersStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import HomeView from '@/views/HomeView.vue'
import LibraryView from '@/views/LibraryView.vue'
import SettingsView from '@/components/SettingsView.vue'
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
import BatchActionBar from '@/components/BatchActionBar.vue'
import SyncTaskToast from '@/components/common/SyncTaskToast.vue'

const tabsStore = useTabsStore()
const settingsStore = useSettingsStore()
const uiStore = useUIStore()
const viewersStore = useViewersStore()
const { showToast } = useToast()
const { t } = useI18n()

onMounted(async () => {
  await tabsStore.refreshData()
  await settingsStore.loadSettings()

  // Start WebDAV status monitoring if enabled
  if (settingsStore.settings.webdavEnabled) {
    settingsStore.startWebDAVStatusCheck()
  }

  // Event listeners
  window.runtime.EventsOn('tab-updated', (updatedTab: any) => {
    // Use in-place update to preserve scroll position and avoid full refresh
    if (updatedTab && updatedTab.id) {
      tabsStore.updateTabInPlace(updatedTab.id, updatedTab)
    } else {
      // Fallback to full refresh if no tab data provided
      tabsStore.refreshData()
    }
  })

  window.runtime.EventsOn('cover-error', (msg: string) => {
    showToast(msg, 'error')
  })

  window.runtime.EventsOn('sync-complete', (msg: string) => {
    showToast(msg, 'info')
    tabsStore.refreshData()
  })

  window.runtime.EventsOn('file-changes-detected', (msg: string) => {
    showToast(msg + ' - ' + t('toast.clickSyncToUpdate'), 'info')
  })

  window.runtime.EventsOn('migration-completed', (target: string) => {
    tabsStore.refreshData()
    // For covers migration, we might need to clear browser cache, but simple reload is enough
    if (target === 'covers') {
      window.location.reload()
    }
  })

  // Listen for cloud download completion (handle toast globally to avoid duplicates)
  window.runtime.EventsOn('cloud-download-single', (data: any) => {
    if (data.status === 'complete') {
      showToast(t('cloud.downloadSuccess'), 'success')
      tabsStore.refreshData()
    } else if (data.status === 'error') {
      showToast(t('cloud.downloadFailed') + ': ' + data.error, 'error')
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
