<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  trackCount: number
  measureCount: number
  isOpen: boolean
  markers?: Array<{ name: string; bar: number }>
  isSelectionActive?: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle'): void
  (e: 'jump-to-bar', bar: number): void
  (e: 'toggle-selection-mode'): void
  (e: 'clear-selection'): void
}>()

const searchBar = ref('')
const markerSearch = ref('')
const activeTab = ref<'navigate' | 'markers'>('navigate')

// Filter markers based on search
const filteredMarkers = computed(() => {
  if (!props.markers || props.markers.length === 0) return []
  if (!markerSearch.value) return props.markers
  const query = markerSearch.value.toLowerCase()
  return props.markers.filter(m => m.name.toLowerCase().includes(query))
})

// Quick jump buttons for common positions
const quickJumps = computed(() => {
  if (props.measureCount <= 0) return []
  const jumps: { label: string; bar: number }[] = []
  jumps.push({ label: 'Start', bar: 1 })
  if (props.measureCount > 8) jumps.push({ label: '¼', bar: Math.floor(props.measureCount / 4) })
  if (props.measureCount > 4) jumps.push({ label: '½', bar: Math.floor(props.measureCount / 2) })
  if (props.measureCount > 8) jumps.push({ label: '¾', bar: Math.floor((props.measureCount * 3) / 4) })
  jumps.push({ label: 'End', bar: props.measureCount })
  return jumps
})

function onSearch() {
  const bar = parseInt(searchBar.value)
  if (!isNaN(bar) && bar > 0 && bar <= props.measureCount) {
    emit('jump-to-bar', bar)
    searchBar.value = ''
  }
}

function jumpToMarker(bar: number) {
  emit('jump-to-bar', bar)
}

function onQuickJump(bar: number) {
  emit('jump-to-bar', bar)
}
</script>

<template>
  <div class="gp-sidebar" :class="{ open: isOpen }">
    <div class="sidebar-toggle" @click="emit('toggle')" title="Toggle Sidebar (T)">
      <span :class="isOpen ? 'icon-chevron-left' : 'icon-chevron-right'"></span>
    </div>

    <Transition name="sidebar-content">
      <div class="sidebar-content" v-if="isOpen">
        <div class="sidebar-header">
          <h3>Score Tools</h3>
          <span class="shortcut-hint">T</span>
        </div>
        
        <!-- Tab Switcher -->
        <div class="tab-switcher" v-if="markers && markers.length > 0">
          <button 
            class="tab-btn" 
            :class="{ active: activeTab === 'navigate' }"
            @click="activeTab = 'navigate'"
          >
            <span class="icon-search"></span>
            Navigate
          </button>
          <button 
            class="tab-btn" 
            :class="{ active: activeTab === 'markers' }"
            @click="activeTab = 'markers'"
          >
            <span class="icon-pin"></span>
            Markers
          </button>
        </div>

        <!-- Navigate Tab -->
        <div class="tool-sections" v-show="activeTab === 'navigate'">
          <div class="tool-section">
            <label>
              <span class="icon-search icon-sm"></span>
              Jump to Measure
            </label>
            <div class="input-group">
              <input 
                v-model="searchBar" 
                type="number" 
                min="1" 
                :max="measureCount"
                :placeholder="`1-${measureCount}`"
                @keyup.enter="onSearch"
              />
              <button @click="onSearch" class="btn-sm" :disabled="!searchBar">
                <span class="icon-chevron-right"></span>
              </button>
            </div>
          </div>

          <!-- Quick Jump Buttons -->
          <div class="tool-section" v-if="quickJumps.length > 0">
            <label>Quick Jump</label>
            <div class="quick-jumps">
              <button 
                v-for="jump in quickJumps" 
                :key="jump.bar"
                class="quick-jump-btn"
                @click="onQuickJump(jump.bar)"
                :title="`Bar ${jump.bar}`"
              >
                {{ jump.label }}
              </button>
            </div>
          </div>

          <div class="tool-section">
            <label>
              <span class="icon-select icon-sm"></span>
              Section Selection
            </label>
            <button 
              class="btn-block" 
              :class="{ active: isSelectionActive }"
              @click="emit('toggle-selection-mode')"
            >
              <span class="icon-select"></span> 
              {{ isSelectionActive ? 'Selection Active' : 'Select Region' }}
            </button>
            <button 
              v-if="isSelectionActive" 
              class="btn-block btn-secondary"
              @click="emit('clear-selection')"
            >
              <span class="icon-close"></span>
              Clear Selection
            </button>
            <p class="help-text">
              <kbd>Click + Drag</kbd> on the score to select a section for looped playback.
            </p>
          </div>
        </div>

        <!-- Markers Tab -->
        <div class="tool-sections" v-show="activeTab === 'markers' && markers && markers.length > 0">
          <div class="tool-section">
            <div class="input-group">
              <span class="icon-search input-icon"></span>
              <input 
                v-model="markerSearch"
                type="text"
                placeholder="Search markers..."
                class="search-input"
              />
            </div>
          </div>
          
          <div class="markers-list">
            <button 
              v-for="marker in filteredMarkers" 
              :key="marker.bar"
              class="marker-item"
              @click="jumpToMarker(marker.bar)"
            >
              <span class="marker-name">{{ marker.name }}</span>
              <span class="marker-bar">Bar {{ marker.bar }}</span>
            </button>
            <p v-if="filteredMarkers.length === 0" class="no-markers">
              No markers found
            </p>
          </div>
        </div>

        <!-- Keyboard Shortcuts -->
        <div class="shortcuts-section">
          <details>
            <summary>
              <span class="icon-settings icon-sm"></span>
              Keyboard Shortcuts
            </summary>
            <div class="shortcuts-list">
              <div class="shortcut-row">
                <kbd>Space</kbd>
                <span>Play / Pause</span>
              </div>
              <div class="shortcut-row">
                <kbd>S</kbd>
                <span>Stop</span>
              </div>
              <div class="shortcut-row">
                <kbd>M</kbd>
                <span>Metronome</span>
              </div>
              <div class="shortcut-row">
                <kbd>↑/↓</kbd>
                <span>Scroll</span>
              </div>
              <div class="shortcut-row">
                <kbd>+/-</kbd>
                <span>Tempo</span>
              </div>
            </div>
          </details>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.gp-sidebar {
  position: relative;
  width: 0;
  height: 100%;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  transition: width 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  flex-shrink: 0;
  z-index: 10;
  overflow: hidden;
}

.gp-sidebar.open {
  width: 280px;
}

.sidebar-toggle {
  position: absolute;
  top: 50%;
  right: -28px;
  width: 28px;
  height: 56px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-left: none;
  border-radius: 0 12px 12px 0;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transform: translateY(-50%);
  color: var(--text-secondary);
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
}

.sidebar-toggle:hover {
  color: var(--primary-color);
  background: var(--bg-hover);
  transform: translateY(-50%) translateX(2px);
}

/* Sidebar content transition */
.sidebar-content-enter-active {
  transition: opacity 0.3s ease 0.1s, transform 0.3s ease 0.1s;
}
.sidebar-content-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.sidebar-content-enter-from,
.sidebar-content-leave-to {
  opacity: 0;
  transform: translateX(-10px);
}

.sidebar-content {
  padding: 1.25rem;
  width: 280px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 0.75rem;
  border-bottom: 2px solid var(--primary-color);
}

.sidebar-header h3 {
  margin: 0;
  font-size: 1.1rem;
  color: var(--text-primary);
  font-weight: 600;
}

.shortcut-hint {
  background: var(--bg-tertiary);
  color: var(--text-muted);
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 0.75rem;
  font-family: monospace;
  border: 1px solid var(--border-color);
}

/* Tab Switcher */
.tab-switcher {
  display: flex;
  gap: 4px;
  background: var(--bg-tertiary);
  padding: 4px;
  border-radius: 8px;
}

.tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 0.85rem;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.tab-btn:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.tab-btn.active {
  background: var(--primary-color);
  color: white;
  box-shadow: 0 2px 8px rgba(150, 82, 51, 0.3);
}

.tool-sections {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  flex: 1;
}

.tool-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.tool-section label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 6px;
}

.input-group {
  display: flex;
  gap: 0.5rem;
  position: relative;
}

.input-group .input-icon {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-muted);
  pointer-events: none;
}

.input-group input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--border-color);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  min-width: 0;
  font-size: 0.9rem;
  transition: all 0.2s;
}

.input-group input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(150, 82, 51, 0.15);
}

.search-input {
  padding-left: 2.25rem !important;
}

.btn-sm {
  padding: 0.5rem 0.75rem;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-sm:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-sm:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Quick Jump Buttons */
.quick-jumps {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.quick-jump-btn {
  flex: 1;
  min-width: 40px;
  padding: 8px 4px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.8rem;
  font-weight: 500;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.quick-jump-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: var(--bg-hover);
  transform: translateY(-2px);
}

.btn-block {
  width: 100%;
  padding: 0.65rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  font-size: 0.9rem;
}

.btn-block:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
}

.btn-block.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
  box-shadow: 0 4px 12px rgba(150, 82, 51, 0.3);
}

.btn-block.btn-secondary {
  background: transparent;
  margin-top: 0.5rem;
}

.btn-block.btn-secondary:hover {
  color: var(--error-color);
  border-color: var(--error-color);
}

.help-text {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin: 0.25rem 0 0 0;
  line-height: 1.4;
}

kbd {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 0.75rem;
  font-family: inherit;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* Markers List */
.markers-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 300px;
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
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.marker-item:hover {
  border-color: var(--primary-color);
  background: var(--bg-hover);
  transform: translateX(4px);
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
}

/* Shortcuts Section */
.shortcuts-section {
  margin-top: auto;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color);
}

.shortcuts-section details {
  cursor: pointer;
}

.shortcuts-section summary {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  color: var(--text-secondary);
  padding: 8px 0;
  list-style: none;
  transition: color 0.2s;
}

.shortcuts-section summary::-webkit-details-marker {
  display: none;
}

.shortcuts-section summary:hover {
  color: var(--text-primary);
}

.shortcuts-section details[open] summary {
  color: var(--primary-color);
  margin-bottom: 8px;
}

.shortcuts-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  animation: fadeSlideIn 0.2s ease;
}

@keyframes fadeSlideIn {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.shortcut-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.shortcut-row span {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.icon-sm {
  width: 0.9em;
  height: 0.9em;
}
</style>
