<script setup lang="ts">
import { useTabsStore, useUIStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'
import { useToast } from '@/composables/useToast'
import TabCard from './TabCard.vue'
import CategoryCard from './CategoryCard.vue'
import BackCard from './BackCard.vue'

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const contextMenu = useContextMenu()
const { showToast } = useToast()

function handleBlankContextMenu(e: MouseEvent) {
  // Only show if not clicking on a card
  if ((e.target as HTMLElement).closest('.tab-card')) return

  e.preventDefault()
  contextMenu.show(e.pageX, e.pageY, [
    { label: 'New Category', action: () => uiStore.showCategoryModal() },
    { label: 'Upload TAB', action: () => addTab(true) },
    { label: 'Link Local TAB', action: () => addTab(false) }
  ])
}

async function addTab(isUpload: boolean) {
  const paths = await window.go.main.App.SelectFiles()
  if (paths && paths.length > 0) {
    let added = 0
    let skipped = 0

    for (const path of paths) {
      try {
        const tabData = await window.go.main.App.ProcessFile(path)
        await window.go.main.App.SaveTab(tabData, isUpload)
        added++
      } catch (err) {
        console.warn('Skipped duplicate or error:', err)
        skipped++
      }
    }

    await tabsStore.refreshData()

    // Show toast
    if (skipped > 0) {
      showToast(`Added ${added} tab(s), ${skipped} skipped (duplicates)`, 'warning')
    } else if (added > 0) {
      showToast(`Added ${added} tab(s)`)
    }
  }
}

function goHome() {
  tabsStore.goHome()
}
</script>

<template>
  <header>
    <h1 @click="goHome" style="cursor: pointer;">Library</h1>
    <div class="actions">
      <button
        id="btn-select-mode"
        class="btn"
        :class="{ active: tabsStore.isBatchSelectMode }"
        @click="tabsStore.toggleBatchSelectMode"
      >
        <span v-if="tabsStore.isBatchSelectMode" class="icon-close"></span>
        <span v-else class="icon-checkbox"></span>
        {{ tabsStore.isBatchSelectMode ? 'Cancel' : 'Select' }}
      </button>
      <button class="btn" @click="addTab(false)">Link Local Tab</button>
      <button class="btn primary" @click="addTab(true)">Upload Tab</button>
    </div>
  </header>

  <div
    id="tab-grid"
    class="tab-grid"
    @contextmenu="handleBlankContextMenu"
  >
    <!-- Back Button -->
    <BackCard v-if="tabsStore.currentCategoryId" />

    <!-- Categories -->
    <CategoryCard
      v-for="category in tabsStore.currentCategories"
      :key="category.id"
      :category="category"
    />

    <!-- Tabs -->
    <TabCard
      v-for="tab in tabsStore.currentTabs"
      :key="tab.id"
      :tab="tab"
    />
  </div>
</template>
