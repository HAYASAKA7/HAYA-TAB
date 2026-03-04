<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useUIStore, useSettingsStore } from '@/stores'
import { useContextMenu } from '@/composables/useContextMenu'
import { useToast } from '@/composables/useToast'
import type { ContextMenuItem } from '@/types'
import TabCard from '@/components/grid/TabCard.vue'
import CategoryCard from '@/components/grid/CategoryCard.vue'
import BackCard from '@/components/grid/BackCard.vue'
import SearchBar from '@/components/common/SearchBar.vue'

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const settingsStore = useSettingsStore()
const contextMenu = useContextMenu()
const { showToast } = useToast()
const viewMode = ref<'singles' | 'categories'>('singles')

const shouldShowFilters = computed(() => {
  if (viewMode.value === 'singles') return true
  // In categories mode, only show filters if inside a specific category (playlist view)
  return !!tabsStore.currentCategoryId
})

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

// Kana row order for Japanese UI Quick Jump Bar
const KANA_ROWS = ['あ', 'か', 'さ', 'た', 'な', 'は', 'ま', 'や', 'ら', 'わ']
const AZ_LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('')

// Check if current UI is Japanese
const isJapaneseUI = computed(() => settingsStore.settings.language === 'ja')

// Get the initial key for a tab based on current UI language
function getInitialKey(tab: typeof tabsStore.tabs[0]): string {
  if (isJapaneseUI.value) {
    // For JA UI: use initialKana (A-Z for Latin, Kana row for Japanese, # for others)
    return tab.initialKana || '#'
  } else {
    // For EN/ZH UI: use initialAz (A-Z for all, # for special chars)
    return tab.initialAz || '#'
  }
}

const groupedTabs = computed(() => {
  const groups: Record<string, typeof tabsStore.tabs> = {}

  // Sort tabs alphabetically first
  const sorted = [...tabsStore.tabs].sort((a, b) => a.title.localeCompare(b.title))

  for (const tab of sorted) {
    const key = getInitialKey(tab)

    if (!groups[key]) {
      groups[key] = []
    }
    groups[key].push(tab)
  }

  // Define the order based on UI language
  let order: string[]
  if (isJapaneseUI.value) {
    // JA UI: A-Z, then Kana rows, then #
    order = [...AZ_LETTERS, ...KANA_ROWS, '#']
  } else {
    // EN/ZH UI: A-Z, then #
    order = [...AZ_LETTERS, '#']
  }

  // Return groups in the defined order
  const orderedGroups: Record<string, typeof tabsStore.tabs> = {}
  for (const key of order) {
    if (groups[key]) {
      orderedGroups[key] = groups[key]
    }
  }

  return orderedGroups
})

// A-Z Quick Jump Bar - follows same order as groupedTabs
const availableLetters = computed(() => {
  return Object.keys(groupedTabs.value)
})

function scrollToLetter(letter: string) {
  document.getElementById(`group-${letter}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// Drag/touch to scroll functionality
const isDragging = ref(false)

function handleAlphabetPointerDown(e: PointerEvent) {
  isDragging.value = true
  ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
  handleAlphabetPointerMove(e)
}

function handleAlphabetPointerMove(e: PointerEvent) {
  if (!isDragging.value) return
  const target = document.elementFromPoint(e.clientX, e.clientY)
  if (target?.classList.contains('alphabet-letter')) {
    const letter = target.getAttribute('data-letter')
    if (letter) scrollToLetter(letter)
  }
}

function handleAlphabetPointerUp() {
  isDragging.value = false
}

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

  const items: ContextMenuItem[] = []

  // Add Upload/Link only if singles mode
  if (viewMode.value === 'singles') {
    items.push(
        { label: t('library.uploadTab'), icon: 'upload', action: () => { addTab(true) } },
        { label: t('library.linkTab'), icon: 'open', action: () => { addTab(false) } }
    )
  }

  // Add New Category only if in root categories view
  if (viewMode.value === 'categories' && !tabsStore.currentCategoryId) {
    items.push({ label: t('library.newCategory'), icon: 'folder', action: () => { uiStore.showCategoryModal() } })
  }

  if (items.length > 0) {
    contextMenu.show(e.pageX, e.pageY, items)
  }
}

async function addTab(isUpload: boolean) {
  const paths = await window.go.app.App.SelectFiles()
  if (paths && paths.length > 0) {
    let added = 0
    let skipped = 0
    const newTabs: any[] = []

    for (const path of paths) {
      try {
        const tabData = await window.go.app.App.ProcessFile(path)
        // If inside a category, pre-assign it
        if (viewMode.value === 'categories' && tabsStore.currentCategoryId) {
            tabData.categoryIds = [tabsStore.currentCategoryId]
        }
        const savedTab = await window.go.app.App.SaveTab(tabData, isUpload)
        if (savedTab) {
          newTabs.push(savedTab)
        }
        added++
      } catch (err) {
        console.warn('Skipped duplicate or error:', err)
        skipped++
      }
    }

    // Add new tabs in-place to preserve scroll position
    if (newTabs.length > 0) {
      tabsStore.addTabsInPlace(newTabs)
    }

    // Show toast
    if (skipped > 0) {
      showToast(t('toast.addedWithSkip', { added, skipped }), 'warning')
    } else if (added > 0) {
      showToast(t('toast.added', { count: added }))
    }
  }
}
</script>

<template>
  <div class="library-view">
    <header class="view-header sticky">
      <h1>{{ t('library.title') }}</h1>
      <div class="toggle-group">
        <button
          class="toggle-btn"
          :class="{ active: viewMode === 'singles' }"
          @click="switchMode('singles')"
        >
          {{ t('library.singles') }}
        </button>
        <button
          class="toggle-btn"
          :class="{ active: viewMode === 'categories' }"
          @click="switchMode('categories')"
        >
          {{ t('library.categories') }}
        </button>
      </div>
      <div class="actions">
        <button
          class="btn icon-btn"
          :class="{ active: tabsStore.isBatchSelectMode }"
          @click="tabsStore.toggleBatchSelectMode"
          :title="t('library.selectMode')"
        >
          <span v-if="tabsStore.isBatchSelectMode" class="icon-close"></span>
          <span v-else class="icon-checkbox"></span>
        </button>
        <button
          v-if="viewMode === 'categories' && !tabsStore.currentCategoryId"
          class="btn"
          @click="uiStore.showCategoryModal()"
          :title="t('library.newCategory')"
        >
          {{ t('library.newCat') }}
        </button>
        <button
          v-if="viewMode === 'singles' && settingsStore.settings.webdavEnabled"
          class="btn icon-btn"
          :disabled="!settingsStore.webdavConnected"
          @click="uiStore.showCloudPickerModal()"
          :title="settingsStore.webdavConnected ? t('cloud.title') : t('cloud.offline')"
        >
          <span class="icon-cloud"></span>
        </button>
        <button
          v-if="viewMode === 'singles'"
          class="btn"
          @click="addTab(false)"
          :title="t('library.linkTab')"
        >
          {{ t('library.link') }}
        </button>
        <button
          v-if="viewMode === 'singles'"
          class="btn primary"
          @click="addTab(true)"
          :title="t('library.uploadTab')"
        >
          {{ t('library.upload') }}
        </button>
      </div>
    </header>

    <div class="search-container">
      <SearchBar :show-filters="shouldShowFilters" />
    </div>

    <div class="view-content" @contextmenu="handleBlankContextMenu">
      <!-- Singles View -->
      <div v-if="viewMode === 'singles'" class="singles-container">
        <div v-if="tabsStore.loading" class="loading-state">{{ t('library.loading') }}</div>
        <div v-else-if="tabsStore.tabs.length === 0" class="empty-state">{{ t('library.noTabs') }}</div>

        <template v-else>
          <div v-for="(group, letter) in groupedTabs" :key="letter" class="letter-group">
            <div :id="`group-${letter}`" class="group-header">{{ letter }}</div>
            <div class="tab-grid">
              <TabCard v-for="tab in group" :key="tab.id" :tab="tab" />
            </div>
          </div>

          <!-- A-Z Quick Jump Bar -->
          <div
            v-if="availableLetters.length > 1"
            class="alphabet-bar"
            @pointerdown="handleAlphabetPointerDown"
            @pointermove="handleAlphabetPointerMove"
            @pointerup="handleAlphabetPointerUp"
            @pointercancel="handleAlphabetPointerUp"
          >
            <span
              v-for="letter in availableLetters"
              :key="letter"
              :data-letter="letter"
              class="alphabet-letter"
              @click="scrollToLetter(letter)"
            >{{ letter }}</span>
          </div>
        </template>
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
            <div v-if="tabsStore.currentCategories.length === 0" class="empty-state">{{ t('library.noCategories') }}</div>
            <div class="tab-grid">
            <CategoryCard
                v-for="cat in tabsStore.currentCategories"
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
  scroll-margin-top: 1rem;
}

.tab-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  justify-content: center;
}

.singles-container {
  position: relative;
}

.alphabet-bar {
  position: fixed;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 4px;
  background: var(--bg-secondary);
  border-radius: 12px;
  user-select: none;
  touch-action: none;
  z-index: 100;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
}

.alphabet-letter {
  font-size: 11px;
  font-weight: 500;
  line-height: 1.6;
  padding: 2px 8px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color 0.15s ease, transform 0.1s ease;
}

.alphabet-letter:hover {
  color: var(--primary-color);
  transform: scale(1.2);
}

.loading-state, .empty-state {
  text-align: center;
  padding: 4rem;
  color: var(--text-muted);
}
</style>
