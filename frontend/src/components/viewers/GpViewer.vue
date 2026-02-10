<script setup lang="ts">
import { ref, computed, watch, onUnmounted, nextTick } from 'vue'
import { useTabsStore, useSettingsStore } from '@/stores'
import { useToast } from '@/composables/useToast'

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
const api = ref<any>(null)
const isLoaded = ref(false)

// Playback state
const isPlaying = ref(false)
const baseTempo = ref(120)
const currentBpm = ref(120)
const playbackSpeed = ref(1.0)
const metronomeEnabled = ref(false)
const tracks = ref<any[]>([])
const selectedTrack = ref(0)

watch(() => settingsStore.settings.audioDevice, (newId) => {
  updateAudioOutput(newId)
})

async function updateAudioOutput(deviceId: string) {
  if (!api.value) return
  
  // Empty string is default device in Web Audio API
  if (deviceId === 'default') deviceId = ''

  try {
    // Attempt to find AudioContext in AlphaTab instance
    // Check multiple possible locations depending on AlphaTab version/internals
    const player = api.value.player
    let ctx = (api.value as any).audioContext || (player && player.context)

    // Check inside synthesis/renderer if not found on player
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
    })

    api.value.playerStateChanged.on((args: any) => {
      isPlaying.value = args.state === 1
    })
  } catch (e) {
    showToast('Failed to load GP Tab: ' + e, 'error')
    console.error(e)
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
  }
}

function scrollGp(amount: number) {
  if (!containerRef.value) return
  const scrollEl = containerRef.value.querySelector('.gp-scroll-wrapper')
  if (scrollEl) {
    scrollEl.scrollTop += amount
  }
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

    <div class="gp-scroll-wrapper">
      <div class="gp-render-area"></div>
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
}

.gp-scroll-wrapper {
  flex: 1;
  overflow: auto;
}

.gp-render-area {
  min-height: 100%;
}
</style>
