<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore } from '@/stores'
import { useDragDrop } from '@/composables/useDragDrop'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const tabsStore = useTabsStore()
const { draggedItem, endDrag } = useDragDrop()
const { showToast } = useToast()

const isDragOver = ref(false)

const parentId = computed(() => {
  const current = tabsStore.currentCategory
  return current?.parentId || ''
})

function handleClick() {
  tabsStore.goBack()
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
    const moved = await tabsStore.batchMoveTabs(parentId.value)
    showToast(t('contextMenu.movedTabs', { count: moved }))
    return
  }

  if (!draggedItem.value) return

  try {
    if (draggedItem.value.type === 'tab') {
      await tabsStore.moveTab(draggedItem.value.id, parentId.value)
    } else {
      await tabsStore.moveCategory(draggedItem.value.id, parentId.value)
    }
    showToast(t('contextMenu.movedSuccessfully'))
  } catch (err) {
    showToast(t('contextMenu.moveFailed') + ': ' + err, 'error')
  }

  endDrag()
}
</script>

<template>
  <div
    class="tab-card folder back-folder"
    :class="{ 'drag-over': isDragOver }"
    @click="handleClick"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <div class="cover-wrapper">
      <span class="icon-back icon-xl"></span>
    </div>
    <div class="info">
      <div class="title">{{ t('nav.back') }}</div>
    </div>
  </div>
</template>
