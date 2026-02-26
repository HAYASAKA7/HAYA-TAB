<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useSettingsStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { useAlphaTab } from '@/composables/useAlphaTab'
import GpFloatingToolbar from './GpFloatingToolbar.vue'
import GpSelectionMenu from './GpSelectionMenu.vue'
import { EventsOn, EventsOff } from '@/wailsjs/runtime/runtime'

const { t } = useI18n()
const props = defineProps<{
  tabId: string
  visible: boolean
}>()

const tabsStore = useTabsStore()
const settingsStore = useSettingsStore()
const { showToast } = useToast()

const tab = computed(() => tabsStore.getTabById(props.tabId))
const isGp = computed(() => tab.value?.type === 'gp')

// Refs
const containerRef = ref<HTMLElement | null>(null)
const scrollWrapperRef = ref<HTMLElement | null>(null)
const floatingToolbarRef = ref<InstanceType<typeof GpFloatingToolbar> | null>(null)

// Composable
const {
  api,
  isLoaded,
  isSoundFontLoaded,
  loadError,
  isServerError,
  loadProgress,
  isPlaying,
  currentBpm,
  playbackSpeed,
  metronomeEnabled,
  isLooping,
  tracks,
  selectedTrack,
  measureCount,
  markers,
  selectionRange,
  isSelectionActive,
  selectionHighlightStyles,

  initialize,
  destroy,
  load,
  updateAudioOutput,
  playPause,
  stop,
  toggleMetronome,
  setBpm,
  setPlaybackSpeed,
  renderTrack,
  toggleLoop,
  clearSelection: composableClearSelection,
  playSelection,
  handleSelectionChange: composableHandleSelectionChange
} = useAlphaTab(t)

// UI State (Local)
const highlightStyle = ref<any>(null)
const menuVisible = ref(false)
const menuPosition = ref({ x: 0, y: 0 })
const isDraggingSelection = ref(false)
const isShiftDragging = ref(false)
const isSectionPlaybackMode = ref(false)

// Scroll tracking for toolbar auto-dodge
const isScrolling = ref(false)
let scrollTimer: ReturnType<typeof setTimeout> | null = null

function onScrollWrapperScroll() {
  isScrolling.value = true
  if (scrollTimer) clearTimeout(scrollTimer)
  scrollTimer = setTimeout(() => {
    isScrolling.value = false
  }, 300)
}

watch(() => settingsStore.settings.audioDevice, (newId) => {
  updateAudioOutput(newId)
})

// Cloud download event handler (only reload, toast is handled globally in App.vue)
function handleCloudDownload(data: any) {
  if (data.tabId !== props.tabId) return
  if (data.status === 'complete') {
    // Reload the tab after download
    loadGpTab()
  }
}

onMounted(() => {
  EventsOn('cloud-download-single', handleCloudDownload)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  EventsOff('cloud-download-single')
  destroy()
})

async function loadGpTab() {
  if (!tab.value || !containerRef.value) return

  try {
    const scrollElement = containerRef.value.querySelector('.gp-scroll-wrapper') as HTMLElement
    const renderArea = containerRef.value.querySelector('.gp-render-area') as HTMLElement

    if (!renderArea || !scrollElement) return

    initialize(renderArea, scrollElement)

    // Setup listener for metadata write-back
    if (api.value) {
        api.value.scoreLoaded.on((score: any) => {
             // Frontend Reverse Write-back
             if (score && props.tabId) {
                const title = score.title || ''
                const artist = score.artist || ''
                const album = score.album || ''

                if (title || artist || album) {
                  window.go.main.App.UpdateTabMetadata(props.tabId, title, artist, album)
                    .catch((err: any) => {
                      console.warn('Failed to update tab metadata:', err)
                    })
                }
              }
        })

        // Setup Selection Listener manually to integrate with UI logic
        if (api.value.playbackRangeHighlightChanged) {
            api.value.playbackRangeHighlightChanged.on((args: any) => {
                onSelectionChange(args)
            })
        }
    }

    updateAudioOutput(settingsStore.settings.audioDevice)
    const port = await window.go.main.App.GetFileServerPort()
    const url = `http://127.0.0.1:${port}/api/file/${props.tabId}`
    
    await load(url)

  } catch (e) {
    // Error is already displayed in the UI via loadError state
    console.error(e)
  }
}

async function downloadToLocal() {
  if (!tab.value?.isCloud) return
  try {
    showToast(t('cloud.downloadingToLocal'), 'info')
    await window.go.main.App.DownloadCloudTabToLocal(props.tabId)
    // Success/error handling is done via cloud-download-single event listener
  } catch (err) {
    console.error('Failed to download cloud tab:', err)
    // Error toast is handled by event listener, only log here
  }
}

function onSelectionChange(args: any) {
    // Check if selection is cleared or invalid
    if (!args || !args.startBeat || !args.endBeat) {
        // In section playback mode, protect the selection from accidental clears
        if (isSectionPlaybackMode.value) return
        
        menuVisible.value = false
        // Call composable to update its state
        composableHandleSelectionChange(args)
        return
    }

    const startBeat = args.startBeat
    const endBeat = args.endBeat
    const startTick = startBeat.absolutePlaybackStart
    const endTick = endBeat.absolutePlaybackStart + endBeat.playbackDuration

    if (startTick === endTick) {
        if (isSectionPlaybackMode.value) return
        menuVisible.value = false
        composableHandleSelectionChange(args)
        return
    }

    // Update composable state
    composableHandleSelectionChange(args)

    // UI Logic: Shift+drag → section playback mode with toolbar
    if (isShiftDragging.value) {
        isSectionPlaybackMode.value = true
        if (args.endBeatBounds && args.endBeatBounds.visualBounds) {
            const bounds = args.endBeatBounds.visualBounds
            menuPosition.value = {
                x: bounds.x + bounds.w / 2,
                y: bounds.y
            }
            isDraggingSelection.value = true
            menuVisible.value = true
        }
    } else {
        isSectionPlaybackMode.value = false
        menuVisible.value = false
        isDraggingSelection.value = true
    }
}

function onBpmChange() {
  setBpm(currentBpm.value)
}

function onSpeedChange() {
  setPlaybackSpeed(playbackSpeed.value)
}

function onTrackChange() {
  renderTrack(selectedTrack.value)
  nextTick(() => {
      scrollWrapperRef.value?.focus()
  })
}

function scrollGp(amount: number) {
  if (!scrollWrapperRef.value) return
  scrollWrapperRef.value.scrollTop += amount
}

function jumpToBar(barNumber: number) {
    if (!api.value) return

    try {
        if (barNumber < 1 || barNumber > measureCount.value) {
            showToast(t('toast.invalidBarNumber', { max: measureCount.value }), 'error')
            return
        }

        const barIndex = barNumber - 1
        const boundsLookup = api.value.boundsLookup || api.value.renderer?.boundsLookup

        if (!boundsLookup) {
            showToast(t('gpViewer.scoreNotRendered'), 'error')
            return
        }
        
        let visualBounds = null
        if (typeof boundsLookup.findMasterBarByIndex === 'function') {
            const masterBarBounds = boundsLookup.findMasterBarByIndex(barIndex)
            if (masterBarBounds) {
                visualBounds = masterBarBounds.visualBounds || masterBarBounds.realBounds || masterBarBounds.lineAlignedBounds
            }
        }
        
        if (!visualBounds && boundsLookup.staffSystems) {
            for (const system of boundsLookup.staffSystems) {
                if (system.bars) {
                    for (const mb of system.bars) {
                        if (mb.index === barIndex) {
                            visualBounds = mb.visualBounds || mb.realBounds || mb.lineAlignedBounds
                            break
                        }
                    }
                }
                if (visualBounds) break
            }
        }

        if (!visualBounds) {
            showToast(t('gpViewer.cannotLocateBar'), 'error')
            return
        }
        
        if (scrollWrapperRef.value) {
            scrollWrapperRef.value.scrollTo({
                top: visualBounds.y - 100,
                left: visualBounds.x - 50,
                behavior: 'smooth'
            })
        }

        if (api.value.score && api.value.score.masterBars && api.value.score.masterBars[barIndex]) {
            api.value.tickPosition = api.value.score.masterBars[barIndex].start
        }
        
        highlightStyle.value = {
            top: (visualBounds.y - 4) + 'px',
            left: (visualBounds.x - 4) + 'px',
            width: (visualBounds.w + 8) + 'px',
            height: (visualBounds.h + 8) + 'px',
            opacity: 1,
            animation: 'highlightPulse 2s ease-out forwards'
        }
        
        setTimeout(() => {
            highlightStyle.value = null
        }, 2000)

        showToast(t('gpViewer.jumpedToMeasure', { bar: barNumber }), 'success')
    } catch(e) {
        console.error('Jump failed', e)
        showToast(t('gpViewer.failedToNavigate'), 'error')
    }
}

function clearSelection() {
    composableClearSelection()
    isSectionPlaybackMode.value = false
    menuVisible.value = false
    showToast(t('gpViewer.selectionCleared'), 'info')
}

// Menu Actions
function handlePlaySelection() {
    playSelection()
}

function handleToggleLoop() {
    if (toggleLoop()) {
        showToast(t('gpViewer.loopingEnabled'), 'success')
    } else {
        showToast(t('gpViewer.loopingDisabled'), 'info')
    }
}

function setMenuSpeed(speed: number) {
    setPlaybackSpeed(speed)
}

function closeMenu() {
    menuVisible.value = false
}

function handleScrollWrapperMouseDown(e: MouseEvent) {
    isShiftDragging.value = e.shiftKey
    isDraggingSelection.value = false
}

function handleScrollWrapperClick() {
    if (isDraggingSelection.value) {
        isDraggingSelection.value = false
        return
    }

    if (!isSectionPlaybackMode.value && isSelectionActive.value) {
        clearSelection()
    }

    floatingToolbarRef.value?.collapse()

    if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur()
    }

    scrollWrapperRef.value?.focus()
}

function handleScrollWrapperContextMenu(e: MouseEvent) {
    if (!isSelectionActive.value || !selectionRange.value || !scrollWrapperRef.value) return

    e.preventDefault()
    isSectionPlaybackMode.value = true

    const rect = scrollWrapperRef.value.getBoundingClientRect()
    menuPosition.value = {
        x: e.clientX - rect.left + scrollWrapperRef.value.scrollLeft,
        y: e.clientY - rect.top + scrollWrapperRef.value.scrollTop
    }
    menuVisible.value = true
}

function handleKeydown(e: KeyboardEvent) {
  if (!props.visible) return

  const target = e.target as HTMLElement
  if (['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) || target.isContentEditable) {
    return
  }

  const step = 100
  const keys = settingsStore.settings.keyBindings
  const key = e.key.toLowerCase()

  if (key === keys.scrollDown) {
    scrollGp(step)
  } else if (key === keys.scrollUp) {
    scrollGp(-step)
  } else if (key === keys.metronome) {
    toggleMetronome()
  } else if (key === keys.playPause) {
    playPause()
  } else if (key === keys.stop) {
    stop()
  } else if (key === keys.bpmPlus) {
    currentBpm.value += 10
    onBpmChange()
  } else if (key === keys.bpmMinus) {
    currentBpm.value -= 10
    onBpmChange()
  } else if (key === keys.clearSelection || key === 'escape') {
    if (isSectionPlaybackMode.value) {
      menuVisible.value = false
      isSectionPlaybackMode.value = false
    } else if (isSelectionActive.value) {
      clearSelection()
    }
  } else if (key === keys.toggleLoop && selectionRange.value) {
    handleToggleLoop()
  } else if (key === keys.jumpToBar) {
    e.preventDefault()
    floatingToolbarRef.value?.openSearch()
  } else if (key === keys.jumpToStart) {
    jumpToBar(1)
  }
}

watch(() => props.visible, async (newVal) => {
  if (newVal) {
    window.addEventListener('keydown', handleKeydown)
  } else {
    window.removeEventListener('keydown', handleKeydown)
  }

  if (newVal && !api.value && isGp.value && tab.value) {
    await nextTick()
    await new Promise(resolve => setTimeout(resolve, 50))
    await loadGpTab()
  }
  if (newVal) {
    window.dispatchEvent(new Event('resize'))
  }
}, { immediate: true })
</script>

<template>
  <div
    v-if="isGp"
    ref="containerRef"
    :id="`gp-view-${tabId}`"
    class="view gp-view"
    :class="{ hidden: !visible }"
  >
    <!-- Toolbar -->
    <div class="gp-toolbar">
      <div class="gp-controls">
        <button class="btn-icon" title="Stop" @click="stop">
          <span class="icon-stop"></span>
        </button>
        <button class="btn-icon" title="Play/Pause" @click="playPause">
          <span :class="isPlaying ? 'icon-pause' : 'icon-play'"></span>
        </button>

        <div class="divider"></div>

        <button
          class="btn-icon"
          :class="{ active: metronomeEnabled }"
          title="Metronome"
          @click="toggleMetronome"
        >
          <span class="icon-metronome"></span>
        </button>

        <input
          type="number"
          class="bpm-input"
          min="30"
          max="300"
          v-model.number="currentBpm"
          title="Set Tempo (BPM)"
          @change="onBpmChange"
        />

        <div class="divider"></div>

        <span class="label">Track:</span>
        <select
          class="track-selector"
          v-model.number="selectedTrack"
          @change="onTrackChange"
        >
          <option v-if="!isLoaded" value="-1">Loading...</option>
          <option
            v-for="track in tracks"
            :key="track.index"
            :value="track.index"
          >
            {{ track.name }}
          </option>
        </select>

        <div class="divider"></div>

        <span class="label">Speed:</span>
        <input
          type="range"
          min="0.25"
          max="2.0"
          step="0.25"
          class="speed-slider"
          v-model.number="playbackSpeed"
          @input="onSpeedChange"
        />
        <span class="speed-val">{{ Math.round(playbackSpeed * 100) }}%</span>
      </div>
    </div>

    <!-- Main Content -->
    <div class="gp-main-content">
        <!-- SoundFont Loading Overlay -->
        <Transition name="fade-mask">
          <div v-if="loadError" class="sf-loading-mask error-mask">
            <div class="sf-loading-content error-content">
              <svg class="error-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
              </svg>
              <span class="error-message">{{ loadError }}</span>
              <span v-if="tab?.isCloud && !isServerError" class="error-hint">{{ t('gpViewer.loadErrorHint') }}</span>
              <div class="error-buttons">
                <button class="retry-button" @click="loadGpTab">{{ t('gpViewer.retry') }}</button>
                <button v-if="tab?.isCloud && !isServerError" class="download-button" @click="downloadToLocal">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
                    <path d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z"/>
                  </svg>
                  {{ t('gpViewer.download') }}
                </button>
              </div>
            </div>
          </div>
          <div v-else-if="!isSoundFontLoaded" class="sf-loading-mask">
            <div class="sf-loading-content">
              <div class="sf-spinner"></div>
              <span v-if="!isLoaded">{{ t('library.loading') }}</span>
              <span v-else>Loading SoundFont...</span>
              <div v-if="!isLoaded && loadProgress > 0" class="sf-progress-bar">
                <div class="sf-progress-fill" :style="{ width: loadProgress + '%' }"></div>
              </div>
            </div>
          </div>
        </Transition>
        <div
            class="gp-scroll-wrapper"
            ref="scrollWrapperRef"
            tabindex="-1"
            @click="handleScrollWrapperClick"
            @mousedown="handleScrollWrapperMouseDown($event)"
            @contextmenu="handleScrollWrapperContextMenu($event)"
            @scroll="onScrollWrapperScroll"
        >
            <div class="gp-render-area"></div>

            <!-- Selection Highlight Overlays (multiple for multi-line selections) -->
            <div 
                v-for="(style, index) in selectionHighlightStyles" 
                :key="index"
                class="selection-highlight" 
                v-show="isSelectionActive" 
                :style="style"
            ></div>
            
            <!-- Jump Highlight Overlay -->
            <div 
                class="highlight-overlay" 
                v-if="highlightStyle" 
                :style="highlightStyle"
            ></div>

            <!-- Context Menu -->
            <GpSelectionMenu 
                :visible="menuVisible"
                :x="menuPosition.x"
                :y="menuPosition.y"
                :isLooping="isLooping"
                :currentSpeed="playbackSpeed"
                :isPlaying="isPlaying"
                @toggle-loop="handleToggleLoop"
                @set-speed="setMenuSpeed"
                @play-selection="handlePlaySelection"
                @close="closeMenu"
            />
        </div>

        <!-- Floating Toolbar -->
        <GpFloatingToolbar
            ref="floatingToolbarRef"
            :measureCount="measureCount"
            :isSelectionActive="isSelectionActive"
            :markers="markers"
            :isScrolling="isScrolling"
            @jump-to-bar="jumpToBar"
            @clear-selection="clearSelection"
        />
    </div>
  </div>
</template>

<style scoped>
.gp-view {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
}

.gp-toolbar {
  flex-shrink: 0;
  z-index: 20;
}

.gp-main-content {
    flex: 1;
    display: flex;
    overflow: hidden;
    position: relative;
}

.gp-scroll-wrapper {
  flex: 1;
  overflow: auto;
  position: relative; /* Context for absolute children */
  outline: none; /* Remove focus outline - we handle focus visually elsewhere */
}

.gp-scroll-wrapper:focus {
  /* Subtle focus indicator */
  box-shadow: inset 0 0 0 2px rgba(150, 82, 51, 0.1);
}

.gp-render-area {
  min-height: 100%;
}

/* AlphaTab built-in cursor styling */
.gp-render-area :deep(.at-cursor-bar) {
    /* Fill the bar with a subtle highlight */
    background: rgba(150, 82, 51, 0.1) !important;
}

.gp-render-area :deep(.at-cursor-beat) {
    /* Beat cursor - thick vertical line for visibility */
    background: linear-gradient(
        180deg,
        var(--primary-color, #965233) 0%,
        color-mix(in srgb, var(--primary-color, #965233), #ff6b3d 50%) 50%,
        var(--primary-color, #965233) 100%
    ) !important;
    width: 12px !important;
    border-radius: 6px;
    box-shadow: 
        0 0 10px rgba(150, 82, 51, 0.8),
        0 0 20px rgba(150, 82, 51, 0.4);
    animation: cursorGlow 1s ease-in-out infinite alternate;
}

@keyframes cursorGlow {
    0% {
        box-shadow: 
            0 0 8px rgba(150, 82, 51, 0.6),
            0 0 16px rgba(150, 82, 51, 0.3);
    }
    100% {
        box-shadow: 
            0 0 12px rgba(150, 82, 51, 0.9),
            0 0 24px rgba(150, 82, 51, 0.5);
    }
}

/* Selection highlight for selected sections */
.selection-highlight {
    position: absolute;
    background: rgba(150, 82, 51, 0.15);
    pointer-events: none;
    border-radius: 4px;
    z-index: 4;
    border: 2px solid rgba(150, 82, 51, 0.5);
    box-shadow: 
        0 0 0 1px rgba(150, 82, 51, 0.3),
        inset 0 0 20px rgba(150, 82, 51, 0.1);
    animation: selectionPulse 2s ease-in-out infinite;
}

@keyframes selectionPulse {
    0%, 100% {
        border-color: rgba(150, 82, 51, 0.5);
        box-shadow: 
            0 0 0 1px rgba(150, 82, 51, 0.3),
            inset 0 0 20px rgba(150, 82, 51, 0.1);
    }
    50% {
        border-color: rgba(150, 82, 51, 0.8);
        box-shadow: 
            0 0 8px rgba(150, 82, 51, 0.4),
            inset 0 0 30px rgba(150, 82, 51, 0.15);
    }
}

/* Jump highlight overlay */
.highlight-overlay {
    position: absolute;
    background: linear-gradient(135deg, rgba(255, 200, 50, 0.4), rgba(255, 150, 0, 0.3));
    pointer-events: none;
    border-radius: 6px;
    z-index: 5;
    box-shadow: 
        0 0 0 2px rgba(255, 180, 0, 0.6),
        0 0 20px rgba(255, 180, 0, 0.4),
        inset 0 0 30px rgba(255, 255, 255, 0.2);
}

@keyframes highlightPulse {
    0% {
        opacity: 1;
        transform: scale(1);
        box-shadow:
            0 0 0 2px rgba(255, 180, 0, 0.8),
            0 0 30px rgba(255, 180, 0, 0.6),
            inset 0 0 30px rgba(255, 255, 255, 0.3);
    }
    25% {
        opacity: 0.9;
        transform: scale(1.02);
        box-shadow:
            0 0 0 3px rgba(255, 180, 0, 0.6),
            0 0 40px rgba(255, 180, 0, 0.4),
            inset 0 0 20px rgba(255, 255, 255, 0.2);
    }
    50% {
        opacity: 0.8;
        transform: scale(1);
    }
    100% {
        opacity: 0;
        transform: scale(0.98);
        box-shadow:
            0 0 0 0 transparent,
            0 0 0 transparent;
    }
}

/* SoundFont Loading Mask */
.sf-loading-mask {
    position: absolute;
    inset: 0;
    z-index: 200;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(2px);
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: all;
}

.sf-loading-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    color: var(--text-secondary, #aaa);
    font-size: 0.95rem;
    font-weight: 500;
}

.sf-spinner {
    width: 36px;
    height: 36px;
    border: 3px solid rgba(150, 82, 51, 0.25);
    border-top-color: var(--primary-color, #965233);
    border-radius: 50%;
    animation: sfSpin 0.8s linear infinite;
}

@keyframes sfSpin {
    to { transform: rotate(360deg); }
}

.sf-progress-bar {
    width: 180px;
    height: 4px;
    background: rgba(150, 82, 51, 0.2);
    border-radius: 2px;
    overflow: hidden;
    margin-top: 4px;
}

.sf-progress-fill {
    height: 100%;
    background: var(--primary-color, #965233);
    border-radius: 2px;
    transition: width 0.15s ease-out;
}

.fade-mask-enter-active {
    transition: opacity 0.2s ease;
}
.fade-mask-leave-active {
    transition: opacity 0.4s ease;
}
.fade-mask-enter-from,
.fade-mask-leave-to {
    opacity: 0;
}

/* Error State */
.error-mask {
    background: rgba(0, 0, 0, 0.6);
}

.error-content {
    text-align: center;
    max-width: 400px;
    padding: 24px;
}

.error-icon {
    width: 48px;
    height: 48px;
    color: var(--danger, #e74c3c);
    margin-bottom: 8px;
}

.error-message {
    color: var(--text-primary, #fff);
    font-size: 1rem;
    line-height: 1.5;
    display: block;
    margin-bottom: 8px;
}

.error-hint {
    color: var(--text-secondary, #aaa);
    font-size: 0.85rem;
    display: block;
    margin-bottom: 16px;
}

.error-buttons {
    display: flex;
    gap: 12px;
    justify-content: center;
}

.retry-button,
.download-button {
    padding: 8px 24px;
    background: var(--primary-color, #965233);
    color: white;
    border: none;
    border-radius: 6px;
    font-size: 0.9rem;
    cursor: pointer;
    transition: background 0.2s ease;
    display: flex;
    align-items: center;
    gap: 6px;
}

.retry-button:hover,
.download-button:hover {
    background: var(--primary-hover, #7a4329);
}

.download-button svg {
    flex-shrink: 0;
}
</style>