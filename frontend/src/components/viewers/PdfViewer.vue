<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useTabsStore, useSettingsStore } from '@/stores'

const props = defineProps<{
  tabId: string
  visible: boolean
}>()

const tabsStore = useTabsStore()
const settingsStore = useSettingsStore()

const tab = computed(() => tabsStore.getTabById(props.tabId))
const iframeRef = ref<HTMLIFrameElement | null>(null)
const viewerUrl = ref('')
const blobUrl = ref('')

// Only render PDF viewer if tab type is pdf
const isPdf = computed(() => tab.value?.type === 'pdf')

// --- Metronome State ---
const isPlaying = ref(false)
const bpm = ref(120)
const volume = ref(1.0)

// Non-reactive audio internals
let audioCtx: AudioContext | null = null
let nextNoteTime = 0
let timerID: number | null = null

const LOOKAHEAD = 25.0   // ms between scheduler calls
const SCHEDULE_AHEAD = 0.1 // seconds to schedule ahead

function getAudioCtx(): AudioContext {
  if (!audioCtx) {
    audioCtx = new AudioContext()
  }
  return audioCtx
}

function scheduleNote(time: number) {
  const ctx = getAudioCtx()
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()

  osc.connect(gain)
  gain.connect(ctx.destination)

  osc.frequency.value = 1000
  gain.gain.setValueAtTime(volume.value, time)
  gain.gain.exponentialRampToValueAtTime(0.001, time + 0.05)

  osc.start(time)
  osc.stop(time + 0.05)
}

function scheduler() {
  const ctx = getAudioCtx()
  while (nextNoteTime < ctx.currentTime + SCHEDULE_AHEAD) {
    scheduleNote(nextNoteTime)
    nextNoteTime += 60.0 / bpm.value
  }
  timerID = window.setTimeout(scheduler, LOOKAHEAD)
}

function startMetronome() {
  const ctx = getAudioCtx()
  if (ctx.state === 'suspended') {
    ctx.resume()
  }
  nextNoteTime = ctx.currentTime
  scheduler()
  isPlaying.value = true
}

function stopMetronome() {
  if (timerID !== null) {
    clearTimeout(timerID)
    timerID = null
  }
  isPlaying.value = false
}

function toggleMetronome() {
  if (isPlaying.value) {
    stopMetronome()
  } else {
    startMetronome()
  }
}

// --- Metronome SVG icon (matches project icon style) ---
const METRONOME_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M10.8 3.2L6 18H4v3h16v-3h-2L13.2 3.2c-.39-1.29-2.01-1.29-2.4 0zm2.2 2.6L16.2 18H7.8l3.2-12.2zM12 7c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm-1 4.8l-2 5.2 2-2 2 2-2-5.2z"/></svg>`
const STOP_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="16" height="16"><rect fill="currentColor" x="3" y="3" width="10" height="10" rx="1"/></svg>`

function onIframeLoad() {
  if (!iframeRef.value) return
  try {
    const doc = iframeRef.value.contentDocument
    if (!doc) return

    // Apply theme class to iframe body for CSS specificity
    const isLight = document.body.getAttribute('data-theme') === 'light'
    doc.body.classList.add(isLight ? 'theme-light' : 'theme-dark')

    // Attach keyboard listener inside iframe so shortcuts work when iframe has focus
    doc.addEventListener('keydown', handleKeydown)

    const toolbarRight = doc.getElementById('toolbarViewerRight')
    if (!toolbarRight) return

    // Prevent duplicate injection
    if (doc.getElementById('haya-metronome-container')) return

    // Inject styles — use --ht-* variables from custom_viewer.css for theme consistency
    const style = doc.createElement('style')
    style.textContent = `
      .haya-metronome-group {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 0 4px;
        margin-inline-end: 4px;
        border-inline-end: 1px solid var(--ht-border, #3e3e42);
      }
      .haya-btn {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 28px;
        height: 28px;
        border: none;
        border-radius: 2px;
        background: transparent;
        color: var(--ht-text, #fff);
        cursor: pointer;
        padding: 0;
      }
      .haya-btn:hover {
        background: var(--ht-hover, #3e3e42);
        color: var(--ht-text, #fff);
      }
      .haya-btn.toggled {
        background: var(--ht-primary, #965233);
        color: #fff;
      }
      .haya-input {
        width: 48px;
        height: 24px;
        border: 1px solid var(--ht-border, #3e3e42);
        border-radius: 2px;
        background: var(--ht-bg, #1e1e1e);
        color: var(--ht-text, #fff);
        text-align: center;
        font-size: 12px;
        padding: 0 2px;
        -moz-appearance: textfield;
      }
      .haya-input::-webkit-inner-spin-button,
      .haya-input::-webkit-outer-spin-button {
        -webkit-appearance: none;
        margin: 0;
      }
      .haya-input:focus {
        outline: none;
        border-color: var(--ht-primary, #965233);
      }
      .haya-bpm-label {
        font-size: 11px;
        color: var(--ht-text-muted, #aaa);
        user-select: none;
      }
    `
    doc.head.appendChild(style)

    // Build container
    const container = doc.createElement('div')
    container.id = 'haya-metronome-container'
    container.className = 'haya-metronome-group'

    // Toggle button
    const btn = doc.createElement('button')
    btn.id = 'haya-metronome-btn'
    btn.className = 'haya-btn'
    btn.title = 'Toggle Metronome (M)'
    btn.innerHTML = METRONOME_SVG
    btn.onclick = () => toggleMetronome()

    // BPM input
    const input = doc.createElement('input')
    input.id = 'haya-metronome-bpm'
    input.className = 'haya-input'
    input.type = 'number'
    input.min = '20'
    input.max = '300'
    input.value = String(bpm.value)
    input.title = 'BPM'
    // Prevent PDF.js shortcuts from firing while typing
    input.onkeydown = (e: KeyboardEvent) => e.stopPropagation()
    input.onchange = () => {
      const v = Math.max(20, Math.min(300, parseInt(input.value) || 120))
      bpm.value = v
      input.value = String(v)
    }

    // BPM label
    const label = doc.createElement('span')
    label.className = 'haya-bpm-label'
    label.textContent = 'BPM'

    container.appendChild(btn)
    container.appendChild(input)
    container.appendChild(label)

    toolbarRight.insertBefore(container, toolbarRight.firstChild)
  } catch (e) {
    // Ignore cross-origin or DOM access issues
  }
}

onMounted(async () => {
  if (!isPdf.value || !tab.value) return
  await loadPdf()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  stopMetronome()
  if (audioCtx) {
    audioCtx.close()
    audioCtx = null
  }
  // Revoke blob URL to free memory
  if (blobUrl.value) {
    URL.revokeObjectURL(blobUrl.value)
  }
})

async function loadPdf() {
  if (!tab.value) return

  try {
    const port = await window.go.main.App.GetFileServerPort()
    // Use streaming endpoint from local server
    const url = `http://127.0.0.1:${port}/api/file/${props.tabId}`

    // Determine PDF.js Theme (0: Auto, 1: Light, 2: Dark)
    let pdfTheme = 2 // Default Dark
    if (document.body.getAttribute('data-theme') === 'light') {
      pdfTheme = 1
    }

    // Determine Locale
    let appLang = document.documentElement.lang || navigator.language || 'en-US'
    if (appLang === 'en') appLang = 'en-US'

    // Construct URL with Hash Params
    viewerUrl.value = `pdfjs/web/viewer.html?file=${encodeURIComponent(url)}#locale=${appLang}&viewerCssTheme=${pdfTheme}`
  } catch (e) {
    console.error('Failed to load PDF:', e)
  }
}

function scrollPdf(amount: number) {
  if (!iframeRef.value) return
  try {
    const doc = iframeRef.value.contentDocument
    if (doc) {
      const viewerContainer = doc.getElementById('viewerContainer')
      if (viewerContainer) {
        viewerContainer.scrollTop += amount
      }
    }
  } catch (e) {
    // Ignore cross-origin issues
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
    scrollPdf(step)
  } else if (key === keys.scrollUp) {
    scrollPdf(-step)
  } else if (key === keys.metronome) {
    toggleMetronome()
  } else if (key === keys.bpmPlus) {
    bpm.value = Math.min(300, bpm.value + 10)
  } else if (key === keys.bpmMinus) {
    bpm.value = Math.max(20, bpm.value - 10)
  }
}

// Watch for visibility changes to trigger resize
watch(() => props.visible, (newVal) => {
  if (newVal) {
    window.addEventListener('keydown', handleKeydown)
  } else {
    window.removeEventListener('keydown', handleKeydown)
  }

  if (newVal && iframeRef.value) {
    // Trigger resize for PDF.js
    window.dispatchEvent(new Event('resize'))
  }
}, { immediate: true })

// Sync metronome playing state to injected button
watch(isPlaying, (playing) => {
  if (!iframeRef.value) return
  try {
    const doc = iframeRef.value.contentDocument
    if (!doc) return
    const btn = doc.getElementById('haya-metronome-btn')
    if (btn) {
      btn.innerHTML = playing ? STOP_SVG : METRONOME_SVG
      btn.classList.toggle('toggled', playing)
    }
  } catch { /* ignore */ }
})

// Sync BPM value to injected input
watch(bpm, (val) => {
  if (!iframeRef.value) return
  try {
    const doc = iframeRef.value.contentDocument
    if (!doc) return
    const input = doc.getElementById('haya-metronome-bpm') as HTMLInputElement | null
    if (input && input.value !== String(val)) {
      input.value = String(val)
    }
  } catch { /* ignore */ }
})
</script>

<template>
  <div
    v-if="isPdf"
    :id="`pdf-view-${tabId}`"
    class="view pdf-view"
    :class="{ hidden: !visible }"
  >
    <div class="pdf-container">
      <iframe
        v-if="viewerUrl"
        ref="iframeRef"
        :src="viewerUrl"
        class="pdf-frame"
        @load="onIframeLoad"
      ></iframe>
    </div>
  </div>
</template>

<style scoped>
.pdf-view {
  width: 100%;
  height: 100%;
}

.pdf-container {
  width: 100%;
  height: 100%;
}

.pdf-frame {
  width: 100%;
  height: 100%;
  border: none;
}
</style>
