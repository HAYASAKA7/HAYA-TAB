<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Category } from '@/types'
import { useTabsStore, useUIStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'
import { useDragDrop } from '@/composables/useDragDrop'
import { useToast } from '@/composables/useToast'

const props = defineProps<{
  category: Category
}>()

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const contextMenu = useContextMenu()
const { draggedItem, startDrag, endDrag } = useDragDrop()
const { showToast } = useToast()

const isDragOver = ref(false)
const coverUrl = ref('')

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

  contextMenu.show(e.pageX, e.pageY, [
    { label: t('contextMenu.open'), action: () => tabsStore.navigateToCategory(props.category.id) },
    { label: t('contextMenu.rename'), action: () => uiStore.showCategoryModal(props.category) },
    { label: t('contextMenu.deleteCategory'), action: () => confirmDelete() }
  ])
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

function handleDragStart(e: DragEvent) {
  startDrag({ type: 'category', id: props.category.id })
  e.dataTransfer!.effectAllowed = 'move'
  e.stopPropagation()
}

function handleDragEnd() {
  endDrag()
}

function handleDragOver(e: DragEvent) {
  e.preventDefault()
  if (!draggedItem.value) {
    // Check for batch drag
    if (tabsStore.isBatchSelectMode && tabsStore.selectedTabIds.size > 0) {
      isDragOver.value = true
      e.dataTransfer!.dropEffect = 'move'
    }
    return
  }
  if (draggedItem.value.type === 'category' && draggedItem.value.id === props.category.id) return
  isDragOver.value = true
  e.dataTransfer!.dropEffect = 'move'
}

function handleDragLeave() {
  isDragOver.value = false
}

async function handleDrop(e: DragEvent) {
  e.preventDefault()
  isDragOver.value = false

  // Handle batch drag
  if (tabsStore.isBatchSelectMode && tabsStore.selectedTabIds.size > 0) {
    const moved = await tabsStore.batchMoveTabs(props.category.id)
    showToast(t('contextMenu.movedTabs', { count: moved }))
    return
  }

  if (!draggedItem.value) return
  if (draggedItem.value.type === 'category' && draggedItem.value.id === props.category.id) return

  try {
    if (draggedItem.value.type === 'tab') {
      await tabsStore.moveTab(draggedItem.value.id, props.category.id)
    } else {
      await tabsStore.moveCategory(draggedItem.value.id, props.category.id)
    }
    showToast(t('contextMenu.movedSuccessfully'))
  } catch (err) {
    showToast(t('contextMenu.moveFailed') + ': ' + err, 'error')
  }

  endDrag()
}

function handleEditClick(e: Event) {
  e.stopPropagation()
  uiStore.showCategoryModal(props.category)
}
</script>

<template>
  <div
    class="tab-card folder"
    :class="{ 'drag-over': isDragOver }"
    draggable="true"
    @click="handleClick"
    @contextmenu="handleContextMenu"
    @dragstart="handleDragStart"
    @dragend="handleDragEnd"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <!-- Edit button -->
    <div
      v-if="!tabsStore.isBatchSelectMode"
      class="edit-btn"
      @click="handleEditClick"
    >
      <span class="icon-edit"></span>
    </div>

    <div class="cover-wrapper">
      <div v-if="coverUrl" class="placeholder-cover">
        <img :src="coverUrl" class="cover-img" loading="lazy" />
      </div>
      <span v-else class="icon-folder icon-xl"></span>
    </div>
    <div class="info">
      <div class="title">{{ category.name }}</div>
    </div>
  </div>
</template>
