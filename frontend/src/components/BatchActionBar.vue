<script setup lang="ts">
import { computed } from 'vue'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showToast } = useToast()

const selectedCount = computed(() => tabsStore.selectedTabIds.size)
const isVisible = computed(() => tabsStore.isBatchSelectMode && selectedCount.value > 0)

async function handleDelete() {
  const selectedTabs = tabsStore.selectedTabs
  const managedCount = selectedTabs.filter(t => t.isManaged).length
  const linkedCount = selectedTabs.length - managedCount

  let message = `You are about to remove <strong>${selectedCount.value}</strong> tab(s).`

  if (managedCount > 0 && linkedCount > 0) {
    message += `<ul>
      <li><strong>${managedCount}</strong> uploaded tab(s) will be <span class="warning-text">deleted permanently</span></li>
      <li><strong>${linkedCount}</strong> linked tab(s) will be unlinked (files remain on disk)</li>
    </ul>`
  } else if (managedCount > 0) {
    message += `<br><br>These <strong>${managedCount}</strong> uploaded tab(s) will be <span class="warning-text">deleted permanently</span>.`
  } else {
    message += `<br><br>These <strong>${linkedCount}</strong> linked tab(s) will be unlinked (files remain on disk).`
  }

  uiStore.showConfirmModal('Remove Tabs', message, 'Remove', true, async () => {
    const deleted = await tabsStore.batchDeleteTabs()
    showToast(`Removed ${deleted} tab(s)`)
  })
}

function handleMove() {
  uiStore.showBatchMoveModal()
}
</script>

<template>
  <div
    id="batch-action-bar"
    :class="{ hidden: !isVisible }"
  >
    <div class="batch-info">
      <span id="batch-selected-count">{{ selectedCount }}</span> selected
      <button class="btn small" @click="tabsStore.selectAllTabs">Select All</button>
    </div>
    <div class="batch-actions">
      <button class="btn" @click="handleMove">
        <span class="icon-folder"></span> Move to...
      </button>
      <button class="btn danger" @click="handleDelete">
        <span class="icon-trash"></span> Remove
      </button>
    </div>
  </div>
</template>
