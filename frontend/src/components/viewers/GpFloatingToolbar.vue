<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'

const props = defineProps<{
  measureCount: number
  isSelectionActive: boolean
  markers?: Array<{ name: string; bar: number }>
}>()

const emit = defineEmits<{
  (e: 'jump-to-bar', bar: number): void
  (e: 'clear-selection'): void
}>()

const isExpanded = ref(false)
const activePanel = ref<'search' | 'markers' | null>(null)
const searchValue = ref('')
const markerSearch = ref('')
const searchInputRef = ref<HTMLInputElement | null>(null)

// Expose collapse method to parent
function collapse() {
  isExpanded.value = false
  activePanel.value = null
}

// Open the search/jump panel directly and focus the input
async function openSearch() {
  isExpanded.value = true
  activePanel.value = 'search'
  // Wait for the DOM to update and the transition to complete
  await nextTick()
  // Small delay to ensure the panel transition has started
  setTimeout(() => {
    searchInputRef.value?.focus()
  }, 50)
}

defineExpose({ collapse, openSearch })

// Filter markers based on search
const filteredMarkers = computed(() => {
  if (!props.markers || props.markers.length === 0) return []
  if (!markerSearch.value) return props.markers
  const query = markerSearch.value.toLowerCase()
  return props.markers.filter(m => m.name.toLowerCase().includes(query))
})

function toggleExpand() {
  isExpanded.value = !isExpanded.value
  if (!isExpanded.value) {
    activePanel.value = null
  }
}

function openPanel(panel: 'search' | 'markers') {
  if (activePanel.value === panel) {
    activePanel.value = null
  } else {
    activePanel.value = panel
    isExpanded.value = true
  }
}

function onSearch() {
  const bar = parseInt(searchValue.value)
  if (!isNaN(bar) && bar > 0 && bar <= props.measureCount) {
    emit('jump-to-bar', bar)
    searchValue.value = ''
    activePanel.value = null
  }
}

function jumpToMarker(bar: number) {
  emit('jump-to-bar', bar)
  activePanel.value = null
}

function handleClearSelection() {
  emit('clear-selection')
}

function handleEscape() {
  // Close any open panel and collapse the toolbar
  activePanel.value = null
  isExpanded.value = false
}
</script>

<template>
  <div class="floating-toolbar" :class="{ expanded: isExpanded }">
    <!-- Main Bubble -->
    <div class="toolbar-bubble" @click="toggleExpand">
      <span class="icon-tool"></span>
    </div>

    <!-- Expanded Tools -->
    <Transition name="tools-slide">
      <div v-if="isExpanded" class="toolbar-tools">
        <!-- Search Button -->
        <button 
          class="tool-btn" 
          :class="{ active: activePanel === 'search' }"
          @click.stop="openPanel('search')"
          title="Jump to Measure (G)"
        >
          <span class="icon-search"></span>
        </button>

        <!-- Markers Button (if available) -->
        <button 
          v-if="markers && markers.length > 0"
          class="tool-btn" 
          :class="{ active: activePanel === 'markers' }"
          @click.stop="openPanel('markers')"
          title="Markers"
        >
          <span class="icon-pin"></span>
        </button>

        <!-- Clear Selection Button (if selection active) -->
        <button 
          v-if="isSelectionActive"
          class="tool-btn clear-btn"
          @click.stop="handleClearSelection"
          title="Clear Selection (Esc)"
        >
          <span class="icon-close"></span>
        </button>
      </div>
    </Transition>

    <!-- Search Panel -->
    <Transition name="panel-pop">
      <div v-if="activePanel === 'search'" class="toolbar-panel" @click.stop>
        <div class="panel-header">
          <span>Jump to Measure</span>
          <button class="close-btn" @click="activePanel = null">
            <span class="icon-close"></span>
          </button>
        </div>
        <div class="panel-content">
          <div class="search-input-group">
            <input 
              ref="searchInputRef"
              v-model="searchValue"
              type="number"
              min="1"
              :max="measureCount"
              :placeholder="`1 - ${measureCount}`"
              @keyup.enter="onSearch"
              @keydown.escape.stop="handleEscape"
            />
            <button class="go-btn" @click="onSearch" :disabled="!searchValue">
              <span class="icon-chevron-right"></span>
            </button>
          </div>
          <div class="quick-jumps">
            <button @click="emit('jump-to-bar', 1)">Start</button>
            <button @click="emit('jump-to-bar', Math.floor(measureCount / 4))">¼</button>
            <button @click="emit('jump-to-bar', Math.floor(measureCount / 2))">½</button>
            <button @click="emit('jump-to-bar', Math.floor(measureCount * 3 / 4))">¾</button>
            <button @click="emit('jump-to-bar', measureCount)">End</button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Markers Panel -->
    <Transition name="panel-pop">
      <div v-if="activePanel === 'markers'" class="toolbar-panel markers-panel" @click.stop>
        <div class="panel-header">
          <span>Markers</span>
          <button class="close-btn" @click="activePanel = null">
            <span class="icon-close"></span>
          </button>
        </div>
        <div class="panel-content">
          <input 
            v-model="markerSearch"
            type="text"
            placeholder="Search markers..."
            class="marker-search"
            @keydown.escape.stop="handleEscape"
          />
          <div class="markers-list">
            <button 
              v-for="marker in filteredMarkers" 
              :key="marker.bar"
              class="marker-item"
              @click="jumpToMarker(marker.bar)"
            >
              <span class="marker-name">{{ marker.name }}</span>
              <span class="marker-bar">{{ marker.bar }}</span>
            </button>
            <p v-if="filteredMarkers.length === 0" class="no-markers">
              No markers found
            </p>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.floating-toolbar {
  position: absolute;
  left: 20px;
  top: 20px;
  z-index: 100;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

/* Main Bubble */
.toolbar-bubble {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--primary-color);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 10;
}

.toolbar-bubble:hover {
  transform: scale(1.1);
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.4);
}

.toolbar-bubble span {
  width: 1.4em;
  height: 1.4em;
  transition: transform 0.3s ease;
}

.floating-toolbar.expanded .toolbar-bubble {
  background: var(--bg-secondary);
  border: 2px solid var(--primary-color);
  color: var(--primary-color);
}

.floating-toolbar.expanded .toolbar-bubble span {
  transform: rotate(90deg);
}

/* Tools Container */
.toolbar-tools {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 24px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
}

.tools-slide-enter-active {
  transition: all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.tools-slide-leave-active {
  transition: all 0.15s ease-in;
}

.tools-slide-enter-from,
.tools-slide-leave-to {
  opacity: 0;
  transform: translateY(-10px) scale(0.9);
}

.tool-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.tool-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  transform: scale(1.1);
}

.tool-btn.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.tool-btn.clear-btn:hover {
  background: var(--error-color);
  border-color: var(--error-color);
  color: white;
}

.tool-btn span {
  width: 1.1em;
  height: 1.1em;
}

/* Panels */
.toolbar-panel {
  position: absolute;
  left: 64px;
  top: 0;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.25);
  min-width: 240px;
  overflow: hidden;
}

.panel-pop-enter-active {
  animation: panelPopIn 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.panel-pop-leave-active {
  animation: panelPopOut 0.15s ease-in;
}

@keyframes panelPopIn {
  0% {
    opacity: 0;
    transform: translateX(-10px) scale(0.95);
  }
  100% {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
}

@keyframes panelPopOut {
  0% {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
  100% {
    opacity: 0;
    transform: translateX(-10px) scale(0.95);
  }
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  font-weight: 600;
  color: var(--text-primary);
}

.panel-header .close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 4px;
  display: flex;
  transition: color 0.2s;
}

.panel-header .close-btn:hover {
  color: var(--error-color);
}

.panel-content {
  padding: 12px 16px;
}

/* Search Panel */
.search-input-group {
  display: flex;
  gap: 8px;
}

.search-input-group input {
  flex: 1;
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-size: 1rem;
  min-width: 0;
}

.search-input-group input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(150, 82, 51, 0.15);
}

.go-btn {
  padding: 10px 14px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.go-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.go-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Quick Jumps */
.quick-jumps {
  display: flex;
  gap: 6px;
  margin-top: 12px;
}

.quick-jumps button {
  flex: 1;
  padding: 8px 4px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.85rem;
  font-weight: 500;
  transition: all 0.2s;
}

.quick-jumps button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: var(--bg-hover);
}

/* Markers Panel */
.markers-panel {
  max-height: 400px;
  display: flex;
  flex-direction: column;
}

.markers-panel .panel-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
}

.marker-search {
  width: 100%;
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-size: 0.9rem;
}

.marker-search:focus {
  outline: none;
  border-color: var(--primary-color);
}

.markers-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 250px;
  overflow-y: auto;
}

.marker-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.marker-item:hover {
  border-color: var(--primary-color);
  background: var(--bg-hover);
}

.marker-name {
  font-size: 0.9rem;
  color: var(--text-primary);
  font-weight: 500;
}

.marker-bar {
  font-size: 0.8rem;
  color: var(--text-muted);
  background: var(--bg);
  padding: 2px 8px;
  border-radius: 10px;
}

.no-markers {
  text-align: center;
  color: var(--text-muted);
  font-size: 0.85rem;
  padding: 1rem;
  margin: 0;
}
</style>
