<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useTabsStore } from '@/stores'

const props = defineProps<{
  tabId: string
  visible: boolean
}>()

const tabsStore = useTabsStore()

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
  // Revoke blob URL to free memory
  if (blobUrl.value) {
    URL.revokeObjectURL(blobUrl.value)
  }
})

async function loadPdf() {
  if (!tab.value) return

  try {
    // Get file as base64 from Go backend
    const b64 = await window.go.main.App.GetTabContent(props.tabId)

    // Convert base64 to blob
    const blob = base64ToBlob(b64)
    const url = URL.createObjectURL(blob)
    blobUrl.value = url

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

function base64ToBlob(base64: string, type = 'application/pdf') {
  const binStr = atob(base64)
  const len = binStr.length
  const arr = new Uint8Array(len)
  for (let i = 0; i < len; i++) {
    arr[i] = binStr.charCodeAt(i)
  }
  return new Blob([arr], { type: type })
}

// Watch for visibility changes to trigger resize
watch(() => props.visible, (newVal) => {
  if (newVal && iframeRef.value) {
    // Trigger resize for PDF.js
    window.dispatchEvent(new Event('resize'))
  }
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
