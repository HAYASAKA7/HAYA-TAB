<script setup lang="ts">
import { computed, ref } from 'vue'
import { useTabsStore, useUIStore, useViewersStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'
import { useToast } from '@/composables/useToast'

const props = defineProps<{
  tabId: string
}>()

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const viewersStore = useViewersStore()
const contextMenu = useContextMenu()
const { showToast } = useToast()

const tab = computed(() => tabsStore.getTabById(props.tabId))
const isPinned = computed(() => viewersStore.isPinned(props.tabId))
const isActive = computed(() => {
  const prefix = tab.value?.type === 'pdf' ? 'pdf' : 'gp'
  return uiStore.currentView === `${prefix}-${props.tabId}`
})

// Drag state
const isDragging = ref(false)
const isDragOver = ref(false)

function handleClick() {
  if (!tab.value) return
  const prefix = tab.value.type === 'pdf' ? 'pdf' : 'gp'
  uiStore.switchView(`${prefix}-${props.tabId}`)
}

function handleClose(e: Event) {
  e.stopPropagation()
  viewersStore.closeTab(props.tabId)
  uiStore.switchView('home')
}

function handleContextMenu(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()

  contextMenu.show(e.pageX, e.pageY, [
    {
      label: isPinned.value ? 'Unpin' : 'Pin',
      action: () => {
        viewersStore.togglePin(props.tabId)
        showToast(isPinned.value ? 'Tab unpinned' : 'Tab pinned')
      }
    },
    {
      label: 'Close',
      action: () => {
        viewersStore.closeTab(props.tabId)
        uiStore.switchView('home')
      }
    }
  ])
}

// Drag handlers
function handleDragStart(e: DragEvent) {
  isDragging.value = true
  e.dataTransfer!.effectAllowed = 'move'
  e.dataTransfer!.setData('text/plain', props.tabId)
}

function handleDragEnd() {
  isDragging.value = false
}

function handleDragOver(e: DragEvent) {
  e.preventDefault()
  isDragOver.value = true
  e.dataTransfer!.dropEffect = 'move'
}

function handleDragLeave() {
  isDragOver.value = false
}

function handleDrop(e: DragEvent) {
  e.preventDefault()
  isDragOver.value = false

  const draggedId = e.dataTransfer!.getData('text/plain')
  if (draggedId && draggedId !== props.tabId) {
    const fromIndex = viewersStore.sortedOpenedTabs.indexOf(draggedId)
    const toIndex = viewersStore.sortedOpenedTabs.indexOf(props.tabId)
    if (fromIndex !== -1 && toIndex !== -1) {
      viewersStore.reorderTabs(fromIndex, toIndex)
    }
  }
}
</script>

<template>
  <div
    v-if="tab"
    class="sidebar-item"
    :class="{
      active: isActive,
      pinned: isPinned,
      dragging: isDragging,
      'drag-over': isDragOver
    }"
    :id="`nav-tab-${tabId}`"
    draggable="true"
    @click="handleClick"
    @contextmenu="handleContextMenu"
    @dragstart="handleDragStart"
    @dragend="handleDragEnd"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <span class="icon">
      <span v-if="isPinned" class="icon-pin"></span>
      <span v-else class="icon-document"></span>
    </span>
    <span class="sidebar-label" :title="tab.title">{{ tab.title }}</span>
    <div class="close-tab" @click="handleClose">
      <span class="icon-close"></span>
    </div>
  </div>
</template>

<style scoped>
.sidebar-item {
  cursor: pointer;
}

.sidebar-item.dragging {
  opacity: 0.5;
}

.sidebar-item.drag-over {
  background: var(--hover-bg);
}

.sidebar-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
