<script setup lang="ts">
import { ref, computed, watch, onUnmounted, nextTick, shallowRef, toRaw } from 'vue'
import { useTabsStore, useSettingsStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import GpFloatingToolbar from './GpFloatingToolbar.vue'
import GpSelectionMenu from './GpSelectionMenu.vue'

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
const api = shallowRef<any>(null)
const isLoaded = ref(false)

// UI State
const highlightStyle = ref<any>(null)
const measureCount = ref(0)
const menuVisible = ref(false)
const menuPosition = ref({ x: 0, y: 0 })
const selectionRange = ref<any>(null)
const isSelectionActive = ref(false)
const isDraggingSelection = ref(false)
const isShiftDragging = ref(false)
const isSectionPlaybackMode = ref(false)
const markers = ref<Array<{ name: string; bar: number }>>([])
const selectionHighlightStyles = ref<any[]>([])

// Playback state
const isPlaying = ref(false)
const baseTempo = ref(120)
const currentBpm = ref(120)
const playbackSpeed = ref(1.0)
const metronomeEnabled = ref(false)
const isLooping = ref(false)
const tracks = ref<any[]>([])
const selectedTrack = ref(0)

watch(() => settingsStore.settings.audioDevice, (newId) => {
  updateAudioOutput(newId)
})

async function updateAudioOutput(deviceId: string) {
  if (!api.value) return
  if (deviceId === 'default') deviceId = ''

  try {
    const player = api.value.player
    let ctx = (api.value as any).audioContext || (player && player.context)

    if (!ctx && player) {
       // @ts-ignore
       if (player.synthesis && player.synthesis.audioContext) {
         // @ts-ignore
         ctx = player.synthesis.audioContext
       }
    }

    if (ctx && typeof ctx.setSinkId === 'function') {
      await ctx.setSinkId(deviceId)
    }
  } catch (e) {
    console.warn('Failed to update audio output device', e)
  }
}

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  if (api.value) {
    try {
      api.value.stop()
      api.value.destroy()
    } catch (e) {
      console.error('Error destroying alphaTab:', e)
    }
  }
})

async function loadGpTab() {
  if (!tab.value || !containerRef.value) return

  try {
    const scrollElement = containerRef.value.querySelector('.gp-scroll-wrapper')
    const renderArea = containerRef.value.querySelector('.gp-render-area')

    if (!renderArea || !scrollElement) return

    // @ts-ignore - alphaTab is loaded globally
    api.value = new alphaTab.AlphaTabApi(renderArea, {
      core: {
        fontDirectory: '/alphatab/font/',
        useWorkers: false
      },
      player: {
        enablePlayer: true,
        enableCursor: true,
        soundFont: '/alphatab/soundfont/sonivox.sf2',
        scrollElement: scrollElement
      },
      display: {
        layoutMode: 'page',
        staveProfile: 'default'
      }
    })

    // Apply audio device
    updateAudioOutput(settingsStore.settings.audioDevice)

    // Load from URL
    const port = await window.go.main.App.GetFileServerPort()
    const url = `http://127.0.0.1:${port}/api/file/${props.tabId}`
    api.value.load(url)

    // Event handlers
    api.value.scoreLoaded.on((score: any) => {
      if (score && score.tempo) {
        baseTempo.value = score.tempo
        currentBpm.value = score.tempo
      }
      
      // Get Measure Count
      if (score && score.masterBars) {
        measureCount.value = score.masterBars.length
        
        // Extract markers from masterBars
        markers.value = []
        score.masterBars.forEach((bar: any, index: number) => {
          if (bar.section && bar.section.text) {
            markers.value.push({
              name: bar.section.text,
              bar: index + 1
            })
          }
        })
      }

      tracks.value = []
      if (score && score.tracks && score.tracks.length > 0) {
        score.tracks.forEach((track: any, index: number) => {
          tracks.value.push({
            index,
            name: track.name || `Track ${index + 1}`
          })
        })
        selectedTrack.value = 0
      }

      isLoaded.value = true

      // Frontend Reverse Write-back: Send parsed metadata to backend
      // AlphaTab has already parsed the internal title, artist, album from the binary file
      if (score && props.tabId) {
        const title = score.title || ''
        const artist = score.artist || ''
        const album = score.album || ''
        
        // Only call backend if we have ANY meaningful metadata to send
        if (title || artist || album) {
          window.go.main.App.UpdateTabMetadata(props.tabId, title, artist, album)
            .catch((err: any) => {
              console.warn('Failed to update tab metadata:', err)
            })
        }
      }
    })

    api.value.playerStateChanged.on((args: any) => {
      isPlaying.value = args.state === 1
    })

    // Selection Handling
    // Correct way to listen for selection in AlphaTab API wrapper:
    if (api.value.playbackRangeHighlightChanged) {
        api.value.playbackRangeHighlightChanged.on((args: any) => {
            handleSelectionChange(args)
        })
    }

  } catch (e) {
    showToast('Failed to load GP Tab: ' + e, 'error')
    console.error(e)
  }
}

function handleSelectionChange(args: any) {
    // Check if selection is cleared or invalid
    if (!args || !args.startBeat || !args.endBeat) {
        // In section playback mode, protect the selection from accidental clears
        if (isSectionPlaybackMode.value) return
        menuVisible.value = false
        selectionRange.value = null
        isSelectionActive.value = false
        selectionHighlightStyles.value = []
        return
    }

    const startBeat = args.startBeat
    const endBeat = args.endBeat

    // Calculate ticks
    const startTick = startBeat.absolutePlaybackStart
    const endTick = endBeat.absolutePlaybackStart + endBeat.playbackDuration

    if (startTick === endTick) {
        // In section playback mode, protect the selection from accidental clears
        if (isSectionPlaybackMode.value) return
        menuVisible.value = false
        selectionRange.value = null
        isSelectionActive.value = false
        selectionHighlightStyles.value = []
        return
    }

    selectionRange.value = {
        startTick: startTick,
        endTick: endTick
    }
    isSelectionActive.value = true

    // Calculate selection highlight bounds - handle multi-line selections
    if (args.startBeatBounds && args.endBeatBounds) {
        const startBounds = args.startBeatBounds.visualBounds
        const endBounds = args.endBeatBounds.visualBounds
        
        if (startBounds && endBounds) {
            const styles: any[] = []
            
            // Check if selection spans multiple lines (Y positions differ significantly)
            const isSameLine = Math.abs(startBounds.y - endBounds.y) < 20
            
            if (isSameLine) {
                // Single line selection - simple rectangle
                const minX = Math.min(startBounds.x, endBounds.x)
                const maxX = Math.max(startBounds.x + startBounds.w, endBounds.x + endBounds.w)
                styles.push({
                    left: (minX - 4) + 'px',
                    top: (startBounds.y - 4) + 'px',
                    width: (maxX - minX + 8) + 'px',
                    height: (startBounds.h + 8) + 'px'
                })
            } else {
                // Multi-line selection - need to find all beats in range and group by line
                const boundsLookup = api.value?.boundsLookup || api.value?.renderer?.boundsLookup
                const groups = boundsLookup ? (boundsLookup.staveGroups || boundsLookup.staffSystems) : null
                
                if (groups) {
                    // Collect all beat bounds within the selection range
                    const lineGroups: Map<number, { minX: number; maxX: number; y: number; h: number }> = new Map()
                    
                    for (const group of groups) {
                        if (!group.bars) continue
                        
                        for (const masterBarBounds of group.bars) {
                            if (!masterBarBounds.bars) continue
                            
                            for (const barBounds of masterBarBounds.bars) {
                                if (!barBounds.beats) continue
                                
                                for (const beatBounds of barBounds.beats) {
                                    if (!beatBounds.beat || !beatBounds.visualBounds) continue
                                    
                                    const beat = beatBounds.beat
                                    const beatStart = beat.absolutePlaybackStart
                                    const beatEnd = beatStart + beat.playbackDuration
                                    
                                    // Check if this beat is within selection range
                                    if (beatEnd > startTick && beatStart < endTick) {
                                        const vb = beatBounds.visualBounds
                                        // Group by Y position (line) - round to avoid floating point issues
                                        const lineY = Math.round(vb.y)
                                        
                                        if (lineGroups.has(lineY)) {
                                            const group = lineGroups.get(lineY)!
                                            group.minX = Math.min(group.minX, vb.x)
                                            group.maxX = Math.max(group.maxX, vb.x + vb.w)
                                        } else {
                                            lineGroups.set(lineY, {
                                                minX: vb.x,
                                                maxX: vb.x + vb.w,
                                                y: vb.y,
                                                h: vb.h
                                            })
                                        }
                                    }
                                }
                            }
                        }
                    }
                    
                    // Create highlight style for each line
                    for (const group of lineGroups.values()) {
                        styles.push({
                            left: (group.minX - 4) + 'px',
                            top: (group.y - 4) + 'px',
                            width: (group.maxX - group.minX + 8) + 'px',
                            height: (group.h + 8) + 'px'
                        })
                    }
                } else {
                    // Fallback to simple bounding box if boundsLookup not available
                    const minX = Math.min(startBounds.x, endBounds.x)
                    const maxX = Math.max(startBounds.x + startBounds.w, endBounds.x + endBounds.w)
                    const minY = Math.min(startBounds.y, endBounds.y)
                    const maxY = Math.max(startBounds.y + startBounds.h, endBounds.y + endBounds.h)
                    
                    styles.push({
                        left: (minX - 4) + 'px',
                        top: (minY - 4) + 'px',
                        width: (maxX - minX + 8) + 'px',
                        height: (maxY - minY + 8) + 'px'
                    })
                }
            }
            
            selectionHighlightStyles.value = styles
        }
    }

    // Shift+drag → section playback mode with toolbar
    // Normal drag → visual selection only, no toolbar
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

function playPause() {
  if (api.value) {
    api.value.playPause()
  }
}

function stop() {
  if (api.value) {
    api.value.stop()
  }
}

function toggleMetronome() {
  if (api.value) {
    metronomeEnabled.value = !metronomeEnabled.value
    api.value.metronomeVolume = metronomeEnabled.value ? 1 : 0
  }
}

function onBpmChange() {
  if (!api.value) return

  let val = currentBpm.value
  if (isNaN(val) || val < 20) val = 20
  if (val > 500) val = 500
  currentBpm.value = val

  const ratio = val / baseTempo.value
  api.value.playbackSpeed = ratio
  playbackSpeed.value = ratio
}

function onSpeedChange() {
  if (!api.value) return

  api.value.playbackSpeed = playbackSpeed.value
  currentBpm.value = Math.round(baseTempo.value * playbackSpeed.value)
}

function onTrackChange() {
  if (!api.value || !api.value.score) return

  const trackIndex = selectedTrack.value
  if (trackIndex >= 0 && api.value.score.tracks[trackIndex]) {
    api.value.renderTracks([api.value.score.tracks[trackIndex]])
    
    // Restore focus to scroll wrapper so keyboard shortcuts work immediately
    nextTick(() => {
        scrollWrapperRef.value?.focus()
    })
  }
}

function scrollGp(amount: number) {
  if (!scrollWrapperRef.value) return
  scrollWrapperRef.value.scrollTop += amount
}

function jumpToBar(barNumber: number) {
    if (!api.value) return
    
    try {
        // Validate input
        if (barNumber < 1 || barNumber > measureCount.value) {
            showToast(`Invalid bar number (1-${measureCount.value})`, 'error')
            return
        }

        const barIndex = barNumber - 1
        
        // Get bounds lookup - available directly on the api object since alphaTab 1.5.0
        // Fallback to renderer.boundsLookup for older versions
        const boundsLookup = api.value.boundsLookup || api.value.renderer?.boundsLookup
        
        if (!boundsLookup) {
            showToast('Score not fully rendered yet', 'error')
            return
        }
        
        // Use the correct API method: findMasterBarByIndex
        let visualBounds = null
        
        // Primary method: use findMasterBarByIndex (correct API method)
        if (typeof boundsLookup.findMasterBarByIndex === 'function') {
            const masterBarBounds = boundsLookup.findMasterBarByIndex(barIndex)
            if (masterBarBounds) {
                // MasterBarBounds has visualBounds, realBounds, and lineAlignedBounds
                visualBounds = masterBarBounds.visualBounds || masterBarBounds.realBounds || masterBarBounds.lineAlignedBounds
            }
        }
        
        // Fallback: iterate through staffSystems to find the bar
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
            showToast('Could not locate bar position', 'error')
            return
        }
        
        // Scroll to the bar
        if (scrollWrapperRef.value) {
            scrollWrapperRef.value.scrollTo({
                top: visualBounds.y - 100,
                left: visualBounds.x - 50,
                behavior: 'smooth'
            })
        }

        // Set cursor position to the start of the bar
        if (api.value.score && api.value.score.masterBars && api.value.score.masterBars[barIndex]) {
            api.value.tickPosition = api.value.score.masterBars[barIndex].start
        }
        
        // Visual Highlight with pulse animation
        highlightStyle.value = {
            top: (visualBounds.y - 4) + 'px',
            left: (visualBounds.x - 4) + 'px',
            width: (visualBounds.w + 8) + 'px',
            height: (visualBounds.h + 8) + 'px',
            opacity: 1,
            animation: 'highlightPulse 2s ease-out forwards'
        }
        
        // Cleanup after animation
        setTimeout(() => {
            highlightStyle.value = null
        }, 2000)
        
        showToast(`Jumped to measure ${barNumber}`, 'success')
    } catch(e) {
        console.error('Jump failed', e)
        showToast('Failed to navigate to measure', 'error')
    }
}

function clearSelection() {
    if (!api.value) return

    // Clear selection in AlphaTab
    api.value.isLooping = false
    api.value.playbackRange = null
    selectionRange.value = null
    isSelectionActive.value = false
    isLooping.value = false
    isSectionPlaybackMode.value = false
    menuVisible.value = false
    selectionHighlightStyles.value = []
    showToast('Selection cleared', 'info')
}

// Menu Actions
function playSelection() {
    if (!api.value || !selectionRange.value) return

    // Always stop first to ensure clean state
    api.value.stop()

    // Set the playback range to the selection
    api.value.playbackRange = toRaw(selectionRange.value)

    // Move cursor to start of selection
    api.value.tickPosition = selectionRange.value.startTick

    // Use nextTick to ensure state updates are processed before starting playback
    nextTick(() => {
        if (api.value) {
            api.value.playPause()
        }
    })
}

function toggleLoop() {
    if (!api.value) return

    isLooping.value = !isLooping.value
    if (isLooping.value && selectionRange.value) {
        api.value.playbackRange = toRaw(selectionRange.value)
        api.value.isLooping = true
        showToast('Looping enabled', 'success')
    } else {
        api.value.isLooping = false
        api.value.playbackRange = null
        isLooping.value = false
        showToast('Looping disabled', 'info')
    }
}

function setMenuSpeed(speed: number) {
    playbackSpeed.value = speed
    onSpeedChange()
}

function closeMenu() {
    menuVisible.value = false
}

function handleScrollWrapperMouseDown(e: MouseEvent) {
    // Track if Shift is held — determines selection mode
    isShiftDragging.value = e.shiftKey
    // User started a potential drag operation - reset flag
    // The flag will be set true by handleSelectionChange when selection completes
    isDraggingSelection.value = false
}

function handleScrollWrapperClick() {
    // After a drag, don't process the click
    if (isDraggingSelection.value) {
        isDraggingSelection.value = false
        return
    }

    // In section playback mode, protect selection from accidental clicks
    // Only Esc key or Clear button can dismiss it
    if (!isSectionPlaybackMode.value && isSelectionActive.value) {
        // Normal selection mode — click blank area clears it
        selectionRange.value = null
        isSelectionActive.value = false
        selectionHighlightStyles.value = []
    }

    // Collapse floating toolbar if expanded
    floatingToolbarRef.value?.collapse()

    // Blur any focused element so shortcuts work
    if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur()
    }

    // Focus the scroll wrapper for keyboard events
    scrollWrapperRef.value?.focus()
}

function handleScrollWrapperContextMenu(e: MouseEvent) {
    // Right-click on an active selection → enter section playback mode
    if (!isSelectionActive.value || !selectionRange.value || !scrollWrapperRef.value) return

    e.preventDefault()
    isSectionPlaybackMode.value = true

    // Calculate position relative to scroll wrapper content (same coordinate space as AlphaTab bounds)
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
      // Single Esc exits section playback mode: close menu, revert to normal selection
      menuVisible.value = false
      isSectionPlaybackMode.value = false
    } else if (isSelectionActive.value) {
      clearSelection()
    }
  } else if (key === keys.toggleLoop && selectionRange.value) {
    // Toggle loop
    toggleLoop()
  } else if (key === keys.jumpToBar) {
    // Open jump-to-bar panel
    e.preventDefault()
    floatingToolbarRef.value?.openSearch()
  } else if (key === keys.jumpToStart) {
    jumpToBar(1)
  }
}

// Watch for visibility changes - initialize alphaTab when visible
watch(() => props.visible, async (newVal) => {
  if (newVal) {
    window.addEventListener('keydown', handleKeydown)
  } else {
    window.removeEventListener('keydown', handleKeydown)
  }

  if (newVal && !api.value && isGp.value && tab.value) {
    // Wait for next tick to ensure DOM is rendered and visible
    await nextTick()
    // Additional small delay to ensure layout is calculated
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
        <div 
            class="gp-scroll-wrapper" 
            ref="scrollWrapperRef" 
            tabindex="-1"
            @click="handleScrollWrapperClick"
            @mousedown="handleScrollWrapperMouseDown($event)"
            @contextmenu="handleScrollWrapperContextMenu($event)"
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
                @toggle-loop="toggleLoop"
                @set-speed="setMenuSpeed"
                @play-selection="playSelection"
                @close="closeMenu"
            />
        </div>

        <!-- Floating Toolbar -->
        <GpFloatingToolbar
            ref="floatingToolbarRef"
            :measureCount="measureCount"
            :isSelectionActive="isSelectionActive"
            :markers="markers"
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
</style>