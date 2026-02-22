<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useTabsStore, useUIStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'
import { useToast } from '@/composables/useToast'
import TabCard from '@/components/grid/TabCard.vue'
import CategoryCard from '@/components/grid/CategoryCard.vue'
import BackCard from '@/components/grid/BackCard.vue'
import SearchBar from '@/components/common/SearchBar.vue'

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const contextMenu = useContextMenu()
const { showToast } = useToast()
const viewMode = ref<'singles' | 'categories'>('singles')

onMounted(async () => {
  if (tabsStore.currentCategoryId) {
    viewMode.value = 'categories'
    // Ensure we are sorted by added_at for playlist view
    tabsStore.setSort('added_at', false)
    await tabsStore.fetchTabsPaginated()
  } else {
    // Setup for Library view
    tabsStore.setSearchScope('global')
    tabsStore.setSort('title', false)
    if (viewMode.value === 'singles') {
        await tabsStore.fetchTabs()
    } else {
        await tabsStore.fetchCategories()
    }
  }
})

// Watch currentCategoryId to refresh tabs if in category mode
watch(() => tabsStore.currentCategoryId, async (newId) => {
  if (newId) {
    // Always switch to categories view if a category is selected
    viewMode.value = 'categories'
    // In playlist mode
    tabsStore.setSearchScope('local')
    tabsStore.setSort('added_at', false) // Default playlist sort
    await tabsStore.fetchTabsPaginated()
  } else if (viewMode.value === 'categories' && !newId) {
    await tabsStore.fetchCategories()
  }
})

const groupedTabs = computed(() => {
  const groups: Record<string, typeof tabsStore.tabs> = {}
  
  // Sort tabs alphabetically first
  const sorted = [...tabsStore.tabs].sort((a, b) => a.title.localeCompare(b.title))

  for (const tab of sorted) {
    const letter = (tab.title[0] || '#').toUpperCase()
    // Group special chars under '#'
    const key = /[A-Z]/.test(letter) ? letter : '#'
    
    if (!groups[key]) {
      groups[key] = []
    }
    groups[key].push(tab)
  }
  
  // Return sorted keys
  const orderedGroups: Record<string, typeof tabsStore.tabs> = {}
  Object.keys(groups).sort().forEach(key => {
    orderedGroups[key] = groups[key]
  })
  
  return orderedGroups
})

function switchMode(mode: 'singles' | 'categories') {
  viewMode.value = mode
  tabsStore.navigateToCategory('') // Reset category
  if (mode === 'singles') {
    tabsStore.setSort('title', false)
    tabsStore.fetchTabs() // Fetch all tabs for grouping
  } else {
    tabsStore.fetchCategories()
  }
}

function handleBlankContextMenu(e: MouseEvent) {
  // Only show if not clicking on a card
  if ((e.target as HTMLElement).closest('.tab-card')) return

  e.preventDefault()
  
  const items = [
    { label: 'Upload TAB', action: () => { addTab(true) } },
    { label: 'Link Local TAB', action: () => { addTab(false) } }
  ]
  
  if (viewMode.value === 'categories') {
    items.unshift({ label: 'New Category', action: () => { uiStore.showCategoryModal() } })
  }

  contextMenu.show(e.pageX, e.pageY, items)
}

async function addTab(isUpload: boolean) {
  const paths = await window.go.main.App.SelectFiles()
  if (paths && paths.length > 0) {
    let added = 0
    let skipped = 0

    for (const path of paths) {
      try {
        const tabData = await window.go.main.App.ProcessFile(path)
        // If inside a category, pre-assign it
        if (viewMode.value === 'categories' && tabsStore.currentCategoryId) {
            tabData.categoryIds = [tabsStore.currentCategoryId]
        }
        await window.go.main.App.SaveTab(tabData, isUpload)
        added++
      } catch (err) {
        console.warn('Skipped duplicate or error:', err)
        skipped++
      }
    }

    if (viewMode.value === 'singles') {
      await tabsStore.fetchTabs()
    } else {
      await tabsStore.refreshData()
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
  <div class="library-view">
    <header class="view-header sticky">
      <h1>Library</h1>
      <div class="toggle-group">
        <button 
          class="toggle-btn" 
          :class="{ active: viewMode === 'singles' }" 
          @click="switchMode('singles')"
        >
          Singles
        </button>
        <button 
          class="toggle-btn" 
          :class="{ active: viewMode === 'categories' }" 
          @click="switchMode('categories')"
        >
          Categories
        </button>
      </div>
      <div class="actions">
        <button
          class="btn icon-btn"
          :class="{ active: tabsStore.isBatchSelectMode }"
          @click="tabsStore.toggleBatchSelectMode"
          title="Select Mode"
        >
          <span v-if="tabsStore.isBatchSelectMode" class="icon-close"></span>
          <span v-else class="icon-checkbox"></span>
        </button>
        <button 
          v-if="viewMode === 'categories'" 
          class="btn" 
          @click="uiStore.showCategoryModal()" 
          title="New Category"
        >
          New Cat
        </button>
        <button class="btn" @click="addTab(false)" title="Link Local Tab">Link</button>
        <button class="btn primary" @click="addTab(true)" title="Upload Tab">Upload</button>
      </div>
    </header>

    <div class="search-container">
      <SearchBar />
    </div>

    <div class="view-content" @contextmenu="handleBlankContextMenu">
      <!-- Singles View -->
      <div v-if="viewMode === 'singles'" class="singles-container">
        <div v-if="tabsStore.loading" class="loading-state">Loading...</div>
        <div v-else-if="tabsStore.tabs.length === 0" class="empty-state">No tabs found.</div>
        
        <div v-else v-for="(group, letter) in groupedTabs" :key="letter" class="letter-group">
          <div class="group-header">{{ letter }}</div>
          <div class="tab-grid">
            <TabCard v-for="tab in group" :key="tab.id" :tab="tab" />
          </div>
        </div>
      </div>

      <!-- Categories View -->
      <div v-else class="categories-container">
        <!-- Playlist View (Tabs inside category) -->
        <div v-if="tabsStore.currentCategoryId" class="playlist-view">
             <div class="tab-grid">
               <BackCard />
               <TabCard v-for="tab in tabsStore.tabs" :key="tab.id" :tab="tab" />
             </div>
             <!-- Infinite scroll trigger could go here -->
        </div>

        <!-- Categories List -->
        <div v-else class="category-list">
            <div v-if="tabsStore.categories.length === 0" class="empty-state">No categories found.</div>
            <div class="tab-grid">
            <CategoryCard 
                v-for="cat in tabsStore.categories" 
                :key="cat.id" 
                :category="cat" 
            />
            </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.library-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.view-header {
  padding: 1.5rem 2rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  z-index: 10;
  position: relative;
}

.view-header h1 {
  margin: 0;
  font-size: 1.8rem;
  min-width: 100px;
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

.actions {
  display: flex;
  gap: 0.5rem;
}

.search-container {
  padding: 1rem 2rem 0;
}

.view-content {
  flex: 1;
  overflow-y: auto;
  padding: 1rem 2rem;
}

.letter-group {
  margin-bottom: 2rem;
}

.group-header {
  font-size: 1.2rem;
  font-weight: bold;
  color: var(--primary);
  margin-bottom: 1rem;
  padding-bottom: 0.5rem;
  position: relative;
  background: transparent;
  z-index: 1;
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
