<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useTabsStore, useUIStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'
import { useToast } from '@/composables/useToast'
import TabCard from '@/components/grid/TabCard.vue'
import CategoryCard from '@/components/grid/CategoryCard.vue'

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const contextMenu = useContextMenu()
const { showToast } = useToast()
const viewMode = ref<'recent' | 'categories'>('recent')

onMounted(async () => {
  // Default to recent view
  await switchMode('recent')
})

// Refresh when returning to Home
watch(() => uiStore.currentView, async (newView) => {
  if (newView === 'home') {
    if (viewMode.value === 'recent') {
      await tabsStore.fetchRecentTabs(20)
    } else {
      await tabsStore.fetchRecentCategories(20)
    }
  }
})

async function switchMode(mode: 'recent' | 'categories') {
  viewMode.value = mode
  if (mode === 'recent') {
    await tabsStore.fetchRecentTabs(20)
  } else {
    await tabsStore.fetchRecentCategories(20)
  }
}

function handleBlankContextMenu(e: MouseEvent) {
  // Only show if not clicking on a card
  if ((e.target as HTMLElement).closest('.tab-card')) return

  e.preventDefault()
  contextMenu.show(e.pageX, e.pageY, [
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

    if (viewMode.value === 'recent') {
      await tabsStore.fetchTabsPaginated()
    }

    // Show toast
    if (skipped > 0) {
      showToast(`Added ${added} tab(s), ${skipped} skipped (duplicates)`, 'warning')
    } else if (added > 0) {
      showToast(`Added ${added} tab(s)`)
    }
  }
}
</script>

<template>
  <div class="home-view">
    <header class="view-header sticky">
      <h1>Home</h1>
      <div class="toggle-group">
        <button 
          class="toggle-btn" 
          :class="{ active: viewMode === 'recent' }" 
          @click="switchMode('recent')"
        >
          Recent
        </button>
        <button 
          class="toggle-btn" 
          :class="{ active: viewMode === 'categories' }" 
          @click="switchMode('categories')"
        >
          Recent Categories
        </button>
      </div>
    </header>

    <div class="view-content" @contextmenu="handleBlankContextMenu">
      <!-- Recent Tabs -->
      <div v-if="viewMode === 'recent'" class="recent-tabs">
        <div v-if="tabsStore.loading" class="loading-state">Loading...</div>
        <div v-else-if="tabsStore.recentTabs.length === 0" class="empty-state">No recent tabs found.</div>
        
        <div v-else class="tab-grid">
          <TabCard v-for="tab in tabsStore.recentTabs" :key="tab.id" :tab="tab" />
        </div>
      </div>

      <!-- Recent Categories -->
      <div v-else class="recent-categories">
        <div v-if="tabsStore.loading" class="loading-state">Loading...</div>
        <div v-else-if="tabsStore.recentCategories.length === 0" class="empty-state">No recent categories found.</div>
        
        <div v-else class="tab-grid">
           <CategoryCard 
             v-for="cat in tabsStore.recentCategories" 
             :key="cat.id" 
             :category="cat" 
           />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.view-header {
  padding: 1.5rem 2rem;
  display: flex;
  align-items: center;
  justify-content: flex-start; /* Changed from space-between */
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  z-index: 10;
  position: relative; /* For absolute positioning of center element */
}

.view-header h1 {
  margin: 0;
  font-size: 1.8rem;
}

.toggle-group {
  display: flex;
  background: var(--bg-tertiary);
  border-radius: 8px;
  padding: 4px;
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
}

.toggle-btn {
  background: transparent;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-muted);
  font-weight: 500;
  transition: all 0.2s;
}

.toggle-btn.active {
  background: var(--primary);
  color: white;
}

.view-content {
  flex: 1;
  overflow-y: auto;
  padding: 1rem 2rem;
}

.tab-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  justify-content: center;
}

.loading-state, .empty-state {
  text-align: center;
  padding: 4rem;
  color: var(--text-muted);
}
</style>
