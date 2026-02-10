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

onMounted(async () => {
  if (!isPdf.value || !tab.value) return
  await loadPdf()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
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
