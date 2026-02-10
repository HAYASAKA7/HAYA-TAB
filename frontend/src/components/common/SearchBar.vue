<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import { useTabsStore } from '@/stores'
import { storeToRefs } from 'pinia'

const tabsStore = useTabsStore()
const { searchQuery, searchFilters, searchScope } = storeToRefs(tabsStore)
const isExpanded = ref(false)
const searchBarRef = ref<HTMLElement | null>(null)

// Debounce search
let timeout: any
const localQuery = ref(searchQuery.value)

watch(localQuery, (newVal) => {
  clearTimeout(timeout)
  timeout = setTimeout(() => {
    tabsStore.setSearchQuery(newVal)
  }, 300)
})

const availableFilters = [
  { label: 'Song Name', value: 'title' },
  { label: 'Artist', value: 'artist' },
  { label: 'Album', value: 'album' },
  { label: 'Tag', value: 'tag' }
]

// Single select for Type
const currentFilterType = computed({
  get: () => searchFilters.value[0] || 'title',
  set: (val: string) => tabsStore.setSearchFilters([val])
})

function handleScopeChange(val: 'local' | 'global') {
    tabsStore.setSearchScope(val)
}

function toggleExpand() {
  isExpanded.value = !isExpanded.value
}

function expand() {
  if (!isExpanded.value) isExpanded.value = true
}

function handleClickOutside(event: MouseEvent) {
  if (isExpanded.value && searchBarRef.value && !searchBarRef.value.contains(event.target as Node)) {
    isExpanded.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div 
    ref="searchBarRef"
    class="search-component" 
    :class="{ expanded: isExpanded }"
  >
    <div class="search-input-container" @click="expand">
      <button class="toggle-btn" @click.stop="toggleExpand">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
      </button>
      <input 
        type="text" 
        v-model="localQuery" 
        placeholder="Search..." 
        @focus="expand"
      />
      <div v-show="isExpanded" class="current-scope-indicator" title="Search Scope">
        {{ searchScope === 'local' ? 'Category' : 'Global' }}
      </div>
    </div>

    <div class="search-filters-edge" :class="{ visible: isExpanded }">
      <div class="filter-group">
        <span class="label">Range:</span>
        <label class="radio-label">
          <input 
            type="radio" 
            name="scope"
            value="local" 
            :checked="searchScope === 'local'"
            @change="handleScopeChange('local')"
          >
          <span>Inside Category</span>
        </label>
        <label class="radio-label">
          <input 
            type="radio" 
            name="scope"
            value="global" 
            :checked="searchScope === 'global'"
            @change="handleScopeChange('global')"
          >
          <span>Global</span>
        </label>
      </div>

      <div class="filter-group">
        <span class="label">Type:</span>
        <label 
          v-for="filter in availableFilters" 
          :key="filter.value" 
          class="radio-label"
        >
          <input 
            type="radio" 
            name="type"
            :value="filter.value"
            v-model="currentFilterType"
          >
          <span>{{ filter.label }}</span>
        </label>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-component {
  margin-bottom: 1rem;
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
}

.search-input-container {
  display: flex;
  align-items: center;
  background: var(--bg);
  padding: 0.5rem 0.8rem;
  cursor: text;
}

.toggle-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  margin-right: 0.5rem;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.toggle-btn:hover {
  background: var(--hover);
  color: var(--text);
}

.search-input-container input {
  border: none;
  background: transparent;
  flex: 1;
  color: var(--text);
  outline: none;
  font-size: 1rem;
  width: 100%;
}

.current-scope-indicator {
  font-size: 0.75rem;
  color: var(--text-muted);
  background: var(--card-bg);
  padding: 2px 6px;
  border-radius: 4px;
  margin-left: 0.5rem;
  text-transform: uppercase;
}

.search-filters-edge {
  max-height: 0;
  opacity: 0;
  padding: 0 0.8rem;
  transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  border-top: 1px solid transparent;
}

.search-filters-edge.visible {
  max-height: 150px; /* Approximate height when expanded */
  opacity: 1;
  padding: 0.8rem;
  border-top-color: var(--border);
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-muted);
  min-width: 50px;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  cursor: pointer;
  font-size: 0.9rem;
  user-select: none;
}

.radio-label input[type="radio"] {
  accent-color: var(--primary);
  margin: 0;
}

.radio-label span {
  color: var(--text);
}
</style>
