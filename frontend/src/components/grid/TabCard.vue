<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import type { Tab, ContextMenuItem } from '@/types'
import { useTabsStore, useUIStore, useViewersStore, useSettingsStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'
import { useToast } from '@/composables/useToast'
import { useDragDrop } from '@/composables/useDragDrop'

const props = defineProps<{
  tab: Tab
}>()

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const viewersStore = useViewersStore()
const settingsStore = useSettingsStore()
const contextMenu = useContextMenu()
const { showToast } = useToast()
const { startDrag, endDrag } = useDragDrop()

const fileServerPort = ref(0)
const coverError = ref(false)
const coverUrl = computed(() => {
  if (!props.tab.coverPath || coverError.value || !fileServerPort.value) return ''
  // Use the file server endpoint for cover images
  return `http://127.0.0.1:${fileServerPort.value}/api/cover/${props.tab.id}?t=${props.tab.addedAt}`
})
const isSelected = computed(() => tabsStore.isTabSelected(props.tab.id))

// Reset error state when cover path changes
watch(() => props.tab.coverPath, () => {
  coverError.value = false
})

onMounted(async () => {
  try {
    fileServerPort.value = await window.go.main.App.GetFileServerPort()
  } catch (e) {
    console.error('Failed to get file server port:', e)
  }
})

function handleClick() {
  if (tabsStore.isBatchSelectMode) {
    tabsStore.toggleTabSelection(props.tab.id)
  } else {
    openTab()
  }
}

async function openTab() {
  const settings = settingsStore.settings

  if (settings.openMethod === 'inner' && props.tab.type === 'pdf') {
    openInternalTab()
  } else if (settings.openGpMethod === 'inner' && props.tab.type === 'gp') {
    openInternalTab()
  } else {
    try {
      await window.go.main.App.OpenTab(props.tab.id)
    } catch (err) {
      console.error(err)
      showToast('Failed to open tab', 'error')
    }
  }
}

async function openInternalTab() {
  try {
    // Notify backend to update timestamp
    await (window.go.main.App as any).MarkAsOpened(props.tab.id)
  } catch (err) {
    console.warn('Failed to mark tab as opened:', err)
  }
  
  viewersStore.openTab(props.tab)
  const prefix = props.tab.type === 'pdf' ? 'pdf' : 'gp'
  uiStore.switchView(`${prefix}-${props.tab.id}`)
}

function handleContextMenu(e: MouseEvent) {
  e.preventDefault()
  e.stopPropagation()

  if (tabsStore.isBatchSelectMode) return

  const items: ContextMenuItem[] = [
    { label: 'Open with System', action: () => window.go.main.App.OpenTab(props.tab.id) },
    { label: 'Open with Inner Viewer', action: () => openInternalTab() },
    { label: 'Edit Metadata', action: () => uiStore.showEditModal(props.tab) },
    { label: 'Add to Category...', action: () => uiStore.showMoveModal(props.tab.id) }
  ]

  if (tabsStore.currentCategoryId) {
    items.push({ 
      label: 'Remove from Category', 
      action: async () => {
        await tabsStore.removeTabFromCategory(props.tab.id, tabsStore.currentCategoryId)
        showToast('Removed from category')
      }
    })
  }

  items.push(
    { label: 'Export TAB', action: () => exportTab() },
    { type: 'separator' },
    { label: props.tab.isManaged ? 'Delete TAB' : 'Unlink TAB', action: () => confirmDelete() }
  )

  contextMenu.show(e.pageX, e.pageY, items)
}

async function exportTab() {
  const dest = await window.go.main.App.SelectFolder()
  if (dest) {
    await window.go.main.App.ExportTab(props.tab.id, dest)
    showToast('Exported')
  }
}

function confirmDelete() {
  const title = props.tab.isManaged ? 'Delete Tab' : 'Unlink Tab'
  const message = props.tab.isManaged
    ? `Are you sure you want to delete "<strong>${props.tab.title}</strong>"?<br><br><span class="warning-text">This will permanently delete the file.</span>`
    : `Are you sure you want to unlink "<strong>${props.tab.title}</strong>"?<br><br>The file will remain on disk.`
  const btnText = props.tab.isManaged ? 'Delete' : 'Unlink'

  uiStore.showConfirmModal(title, message, btnText, true, async () => {
    await tabsStore.deleteTab(props.tab.id)
  })
}

function handleDragStart(e: DragEvent) {
  if (tabsStore.isBatchSelectMode && !isSelected.value) return

  startDrag({ type: 'tab', id: props.tab.id })
  e.dataTransfer!.effectAllowed = 'move'
  e.stopPropagation()
}

function handleDragEnd() {
  endDrag()
}

function handleEditClick(e: Event) {
  e.stopPropagation()
  uiStore.showEditModal(props.tab)
}

function handleCheckboxClick(e: Event) {
  e.stopPropagation()
  tabsStore.toggleTabSelection(props.tab.id)
}
</script>

<template>
  <div
    class="tab-card"
    :class="{ selected: tabsStore.isBatchSelectMode && isSelected }"
    :draggable="!tabsStore.isBatchSelectMode || isSelected"
    @click="handleClick"
    @contextmenu="handleContextMenu"
    @dragstart="handleDragStart"
    @dragend="handleDragEnd"
  >
    <!-- Checkbox for batch mode -->
    <div
      v-if="tabsStore.isBatchSelectMode"
      class="select-checkbox"
      :class="{ checked: isSelected }"
      @click="handleCheckboxClick"
    >
      <span class="icon-checkbox"></span>
    </div>

    <!-- Edit button -->
    <div
      v-if="!tabsStore.isBatchSelectMode"
      class="edit-btn"
      @click="handleEditClick"
    >
      <span class="icon-edit"></span>
    </div>

    <!-- Cover -->
    <div class="cover-wrapper">
      <div class="placeholder-cover">
        <img
          v-if="coverUrl"
          :src="coverUrl"
          class="cover-img"
          loading="lazy"
          @error="coverError = true"
        />
        <span v-else class="icon-music icon-xl"></span>
      </div>
    </div>

    <!-- Info -->
    <div class="info">
      <div class="title" :title="tab.title">{{ tab.title }}</div>
      <div class="artist" :title="tab.artist">{{ tab.artist }}</div>
      <div class="type-badge">{{ tab.type }}</div>
      <div v-if="tab.tag" class="tag-badge" :title="tab.tag">{{ tab.tag }}</div>
    </div>
  </div>
</template>
