<script setup lang="ts">
import { useTabsStore, useUIStore, useViewersStore } from '@/stores'
import SidebarTabItem from './SidebarTabItem.vue'

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const viewersStore = useViewersStore()

function goHome() {
  tabsStore.goHome()
  uiStore.switchView('home')
}

function goSettings() {
  uiStore.switchView('settings')
}

function toggleSidebar() {
  uiStore.toggleSidebar()
}
</script>

<template>
  <aside id="sidebar" :class="{ collapsed: uiStore.sidebarCollapsed }">
    <button id="sidebar-toggle" @click="toggleSidebar">
      <span class="icon-menu"></span>
    </button>
    <div
      id="nav-home"
      class="sidebar-item"
      :class="{ active: uiStore.currentView === 'home' }"
      @click="goHome"
    >
      <span class="icon"><span class="icon-home"></span></span>
      <span class="sidebar-label">Home</span>
    </div>
    <div
      id="nav-settings"
      class="sidebar-item"
      :class="{ active: uiStore.currentView === 'settings' }"
      @click="goSettings"
    >
      <span class="icon"><span class="icon-settings"></span></span>
      <span class="sidebar-label">Settings</span>
    </div>
    <div class="sidebar-divider"></div>
    <div id="opened-tabs-list">
      <SidebarTabItem
        v-for="tabId in viewersStore.sortedOpenedTabs"
        :key="tabId"
        :tab-id="tabId"
      />
    </div>
  </aside>
</template>
