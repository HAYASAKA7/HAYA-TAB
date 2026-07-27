<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { usePlatformStore, useUIStore } from '@/stores'
import type { TopLevelDestination } from '@/types'

const { t } = useI18n()
const platformStore = usePlatformStore()
const uiStore = useUIStore()

const destinations: Array<{
  id: TopLevelDestination
  labelKey: string
  icon: string
}> = [
  { id: 'library', labelKey: 'nav.library', icon: 'icon-library' },
  { id: 'offline', labelKey: 'nav.offline', icon: 'icon-cloud' },
  { id: 'search', labelKey: 'nav.search', icon: 'icon-search' },
  { id: 'settings', labelKey: 'nav.settings', icon: 'icon-settings' },
]
</script>

<template>
  <nav
    v-if="platformStore.capabilities.webTopLevelTabs && !platformStore.capabilities.nativeTopLevelTabs"
    class="mobile-bottom-nav"
    aria-label="Primary"
  >
    <button
      v-for="destination in destinations"
      :key="destination.id"
      type="button"
      class="mobile-nav-item"
      :class="{ active: uiStore.topLevelDestination === destination.id }"
      :aria-current="uiStore.topLevelDestination === destination.id ? 'page' : undefined"
      :aria-label="t(destination.labelKey)"
      @click="uiStore.selectTopLevelDestination(destination.id)"
    >
      <span :class="destination.icon" aria-hidden="true"></span>
      <span class="mobile-nav-label">{{ t(destination.labelKey) }}</span>
    </button>
  </nav>
</template>

<style scoped>
.mobile-bottom-nav {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 120;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  padding: 4px 8px calc(4px + env(safe-area-inset-bottom));
  border-top: 1px solid var(--border);
  background: var(--sidebar-bg);
}

.mobile-nav-item {
  min-width: 44px;
  min-height: 44px;
  padding: 4px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-muted);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  font: inherit;
}

.mobile-nav-item.active {
  color: var(--primary);
  background: var(--hover);
}

.mobile-nav-label {
  max-width: 100%;
  overflow: hidden;
  font-size: 0.7rem;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
