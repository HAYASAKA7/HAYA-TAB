<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Category } from '@/types'
import { SYSTEM_CLOUD_CATEGORY_ID } from '@/types'
import { useTabsStore, useUIStore, useSettingsStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'

const props = defineProps<{
  category: Category
}>()

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const settingsStore = useSettingsStore()
const contextMenu = useContextMenu()

const coverUrl = ref('')

// Check if this is the system cloud category
const isCloudCategory = computed(() => props.category.id === SYSTEM_CLOUD_CATEGORY_ID)

// Check if cloud category is offline
const isCloudOffline = computed(() =>
  isCloudCategory.value && !settingsStore.webdavConnected
)

// Display name with i18n support for system categories
const displayName = computed(() => {
  if (isCloudCategory.value) {
    return t('category.cloudStorage')
  }
  return props.category.name
})

async function loadCover(path: string) {
  if (!path) return
  try {
    const b64 = await window.go.main.App.GetCover(path)
    if (b64) {
      coverUrl.value = `data:image/jpeg;base64,${b64}`
    }
  } catch (e) {
    console.error('Failed to load category cover:', e)
  }
}

watch(() => props.category, (newCat: Category) => {
  const path = newCat.effectiveCoverPath || newCat.coverPath
  if (path) {
    loadCover(path)
  } else {
    coverUrl.value = ''
  }
}, { deep: true, immediate: true })

function handleClick() {
  // Navigate first, then switch view to ensure LibraryView mounts with correct categoryId
  tabsStore.navigateToCategory(props.category.id)
  uiStore.switchView('library')
}

function handleContextMenu(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()

  // Build menu items based on category type
  const menuItems = [
    { label: t('contextMenu.open'), action: () => tabsStore.navigateToCategory(props.category.id) }
  ]

  // Don't allow rename/delete for system cloud category
  if (!isCloudCategory.value) {
    menuItems.push(
      { label: t('contextMenu.rename'), action: () => uiStore.showCategoryModal(props.category) },
      { label: t('contextMenu.deleteCategory'), action: () => confirmDelete() }
    )
  }

  contextMenu.show(e.pageX, e.pageY, menuItems)
}

function confirmDelete() {
  uiStore.showConfirmModal(
    t('contextMenu.deleteCategory'),
    `${t('contextMenu.confirmDeleteCategory', { name: props.category.name })}<br><br>${t('contextMenu.confirmDeleteCategoryInfo')}`,
    t('confirm.delete'),
    true,
    async () => {
      await tabsStore.deleteCategory(props.category.id)
    }
  )
}
</script>

<template>
  <div
    class="tab-card folder"
    :class="{
      'cloud-offline': isCloudOffline
    }"
    @click="handleClick"
    @contextmenu="handleContextMenu"
  >
    <!-- Offline indicator for cloud category -->
    <div v-if="isCloudOffline" class="offline-badge">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="cloud-offline-icon">
        <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM19 18H6c-2.21 0-4-1.79-4-4 0-2.05 1.53-3.76 3.56-3.97l1.07-.11.5-.95C8.08 7.14 9.94 6 12 6c2.62 0 4.88 1.86 5.39 4.43l.3 1.5 1.53.11c1.56.1 2.78 1.41 2.78 2.96 0 1.65-1.35 3-3 3z"/>
      </svg>
      <svg class="offline-slash" viewBox="0 0 24 24" fill="none" stroke="var(--danger)" stroke-width="2.5">
        <line x1="4" y1="4" x2="20" y2="20"/>
      </svg>
    </div>

    <div class="cover-wrapper">
      <!-- Cloud category icon -->
      <div v-if="isCloudCategory" class="cloud-icon-wrapper">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="cloud-icon">
          <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM19 18H6c-2.21 0-4-1.79-4-4 0-2.05 1.53-3.76 3.56-3.97l1.07-.11.5-.95C8.08 7.14 9.94 6 12 6c2.62 0 4.88 1.86 5.39 4.43l.3 1.5 1.53.11c1.56.1 2.78 1.41 2.78 2.96 0 1.65-1.35 3-3 3z"/>
        </svg>
      </div>
      <!-- Regular category cover -->
      <div v-else-if="coverUrl" class="placeholder-cover">
        <img :src="coverUrl" class="cover-img" loading="lazy" />
      </div>
      <span v-else class="icon-folder icon-xl"></span>
    </div>
    <div class="info">
      <div class="title">{{ displayName }}</div>
    </div>
  </div>
</template>

<style scoped>
/* Cloud category specific styles */
.cloud-category .cover-wrapper {
  background: linear-gradient(135deg, var(--bg-secondary) 0%, var(--bg) 100%);
}

.cloud-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.cloud-icon {
  width: 64px;
  height: 64px;
  color: var(--primary);
}

.cloud-offline {
  opacity: 0.6;
}

.cloud-offline .cloud-icon {
  color: var(--text-muted);
}

.offline-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 10;
  background: var(--bg);
  border-radius: 50%;
  padding: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cloud-offline-icon {
  width: 20px;
  height: 20px;
  color: var(--text-muted);
}

.offline-slash {
  position: absolute;
  width: 20px;
  height: 20px;
}
</style>
