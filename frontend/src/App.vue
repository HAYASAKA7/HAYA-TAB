<script setup lang="ts">
import { onMounted } from 'vue'
import { useTabsStore, useSettingsStore, useUIStore, useViewersStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import TabGrid from '@/components/grid/TabGrid.vue'
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
import BatchActionBar from '@/components/BatchActionBar.vue'

const tabsStore = useTabsStore()
const settingsStore = useSettingsStore()
const uiStore = useUIStore()
const viewersStore = useViewersStore()
const { showToast } = useToast()

onMounted(async () => {
  await tabsStore.refreshData()
  await settingsStore.loadSettings()

  // Event listeners
  window.runtime.EventsOn('tab-updated', () => {
    tabsStore.refreshData()
  })

  window.runtime.EventsOn('cover-error', (msg: string) => {
    showToast(msg, 'error')
  })

  window.runtime.EventsOn('sync-complete', (msg: string) => {
    showToast(msg, 'info')
    tabsStore.refreshData()
  })

  window.runtime.EventsOn('file-changes-detected', (msg: string) => {
    showToast(msg + ' - Click Sync to update.', 'info')
  })
})

function isViewActive(viewType: string): boolean {
  if (viewType === 'home') {
    return uiStore.currentView === 'home'
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
        <TabGrid />
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

    <!-- Toast & Context Menu -->
    <Toast />
    <ContextMenu />
  </div>
</template>

<style>
@import '@/assets/style.css';
@import '@/assets/icons.css';
</style>
