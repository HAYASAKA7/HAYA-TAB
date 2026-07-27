<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePlatformStore, useTabsStore, useUIStore, useViewersStore } from '@/stores'
import SidebarTabItem from './SidebarTabItem.vue'

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const viewersStore = useViewersStore()
const platformStore = usePlatformStore()
const isMobileTablet = computed(() => (
  platformStore.isMobile && platformStore.capabilities.formFactor === 'tablet'
))

function goHome() {
  tabsStore.goHome()
  uiStore.switchView('home')
}

function goLibrary() {
  uiStore.switchView('library')
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
      v-if="!isMobileTablet"
      id="nav-home"
      class="sidebar-item"
      :class="{ active: uiStore.currentView === 'home' }"
      @click="goHome"
    >
      <span class="icon"><span class="icon-home"></span></span>
      <span class="sidebar-label">{{ t('nav.home') }}</span>
    </div>
    <div
      v-if="!isMobileTablet"
      id="nav-library"
      class="sidebar-item"
      :class="{ active: uiStore.currentView === 'library' }"
      @click="goLibrary"
    >
      <span class="icon"><span class="icon-library"></span></span>
      <span class="sidebar-label">{{ t('nav.library') }}</span>
    </div>
    <div
      v-if="!isMobileTablet"
      id="nav-settings"
      class="sidebar-item"
      :class="{ active: uiStore.currentView === 'settings' }"
      @click="goSettings"
    >
      <span class="icon settings-icon-container">
        <span class="icon-settings"></span>
        <span v-if="uiStore.updateInfo?.hasUpdate" class="update-badge"></span>
      </span>
      <span class="sidebar-label">{{ t('nav.settings') }}</span>
    </div>
    <div
      v-if="!isMobileTablet && uiStore.hasPlugins"
      id="nav-plugins"
      data-testid="plugins-navigation"
      class="sidebar-item"
      :class="{ active: uiStore.currentView === 'plugins' }"
      @click="uiStore.switchView('plugins')"
    >
      <span class="icon"><span class="icon-plugins"></span></span>
      <span class="sidebar-label">{{ t('nav.plugins') }}</span>
    </div>
    <div v-if="!isMobileTablet" class="sidebar-divider"></div>
    <div id="opened-tabs-list">
      <SidebarTabItem
        v-for="tabId in viewersStore.sortedOpenedTabs"
        :key="tabId"
        :tab-id="tabId"
      />
    </div>
  </aside>
</template>

<style scoped>
.settings-icon-container {
  position: relative;
  display: inline-flex;
}

.update-badge {
  position: absolute;
  top: -2px;
  right: -2px;
  width: 8px;
  height: 8px;
  background-color: var(--accent-color, #e74c3c);
  border-radius: 50%;
  border: 2px solid var(--sidebar-bg);
}
</style>
