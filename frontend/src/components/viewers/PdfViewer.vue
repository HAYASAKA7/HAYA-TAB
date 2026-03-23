
<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useTabsStore, useSettingsStore } from '@/stores'
import { SettingsService, TabService } from '@/services'

const props = defineProps<{ tabId: string; visible: boolean }>()

const tabsStore = useTabsStore()
const settingsStore = useSettingsStore()

const tab = computed(() => tabsStore.getTabById(props.tabId))
const iframeRef = ref<HTMLIFrameElement | null>(null)
const viewerUrl = ref('')
const blobUrl = ref('')
const isPdf = computed(() => tab.value?.type === 'pdf')

const isPlaying = ref(false)
const bpm = ref(120)
const volume = ref(1.0)
let audioCtx: AudioContext | null = null
let nextNoteTime = 0
let timerID: number | null = null

const isAutoScrolling = ref(false)
const scrollSpeed = ref(1)
let scrollFrameId: number | null = null

type AnnotationTool = 'pen' | 'highlight' | 'eraser'
interface AnnotationPoint { x: number; y: number; p?: number }
interface AnnotationStroke { id: string; type: 'pen' | 'highlight'; color: string; lineWidth: number; points: AnnotationPoint[] }
interface PageAnnotation { pageNumber: number; width: number; height: number; strokes: AnnotationStroke[] }

const annotationEnabled = ref(false)
const annotationTool = ref<AnnotationTool>('pen')
const annotationColor = ref('#ff3b30')
const annotationLineWidth = ref(0.005)
const pageAnnotations = new Map<number, PageAnnotation>()
const pageCanvases = new Map<number, HTMLCanvasElement>()
const loadedAnnotationPages = new Set<number>()
const saveTimers = new Map<number, number>()
let pageMutationObserver: MutationObserver | null = null
let viewerResizeObserver: ResizeObserver | null = null
let isAnnotationDrawing = false
let activeStroke: AnnotationStroke | null = null
let activeStrokePageNumber: number | null = null

const LOOKAHEAD = 25.0
const SCHEDULE_AHEAD = 0.1

const METRONOME_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M10.8 3.2L6 18H4v3h16v-3h-2L13.2 3.2c-.39-1.29-2.01-1.29-2.4 0zm2.2 2.6L16.2 18H7.8l3.2-12.2zM12 7c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm-1 4.8l-2 5.2 2-2 2 2-2-5.2z"/></svg>`
const STOP_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="16" height="16"><rect fill="currentColor" x="3" y="3" width="10" height="10" rx="1"/></svg>`
const SCROLL_DOWN_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="16" height="16"><path fill="currentColor" d="M8 2v10M4 8l4 4 4-4"/></svg>`
const SCROLL_PAUSE_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="16" height="16"><rect fill="currentColor" x="4" y="3" width="3" height="10" rx="1"/><rect fill="currentColor" x="9" y="3" width="3" height="10" rx="1"/></svg>`
const ANNO_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="16" height="16"><path fill="currentColor" d="M11.9 1.4a1.5 1.5 0 012.1 2.1l-8.6 8.6-3.2.8.8-3.2 8.9-8.3z"/></svg>`
const PEN_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="14" height="14"><path fill="currentColor" d="M11.6 1.2l3.2 3.2-7.5 7.5H4.1V8.7l7.5-7.5zm-8 11.6h8.8v1.2H3.6v-1.2z"/></svg>`
const HIGHLIGHT_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="14" height="14"><path fill="currentColor" d="M1 10h10v4H1v-4zm3.2-8l6.6 6.6-1.9 1.9-6.6-6.6L4.2 2z"/></svg>`
const ERASER_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="14" height="14"><path fill="currentColor" d="M6.2 2l7.8 7.8-2.2 2.2H4L1 8.9 6.2 2zm-3.4 7L4.5 11h6.8l1.1-1.1L6.2 3.7 2.8 9z"/></svg>`
const UNDO_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="14" height="14"><path fill="currentColor" d="M6.1 3L2 7.1l4.1 4.1.9-.9L4.4 7.8H9c2.2 0 4 1.8 4 4v.2h1v-.2c0-2.8-2.2-5-5-5H4.4L7 3.9 6.1 3z"/></svg>`
const CLEAR_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="14" height="14"><path fill="currentColor" d="M4 4h8v1H4V4zm1 2h6l-.6 7H5.6L5 6zm1-3h4l.6 1H5.4L6 3z"/></svg>`

const getIframeDoc = () => iframeRef.value?.contentDocument ?? null
const generateStrokeID = () => `stroke_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`
function getAudioCtx(): AudioContext {
  if (!audioCtx) audioCtx = new AudioContext()
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
  if (ctx.state === 'suspended') ctx.resume()
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

const toggleMetronome = () => (isPlaying.value ? stopMetronome() : startMetronome())

function startAutoScroll() {
  const doc = getIframeDoc()
  const viewer = doc?.getElementById('viewerContainer') as HTMLElement | null
  if (!viewer) return
  const viewerContainer = viewer
  function step() {
    const maxScroll = viewerContainer.scrollHeight - viewerContainer.clientHeight
    if (viewerContainer.scrollTop >= maxScroll) return stopAutoScroll()
    viewerContainer.scrollTop += scrollSpeed.value * 0.5
    scrollFrameId = requestAnimationFrame(step)
  }
  scrollFrameId = requestAnimationFrame(step)
  isAutoScrolling.value = true
}

function stopAutoScroll() {
  if (scrollFrameId !== null) {
    cancelAnimationFrame(scrollFrameId)
    scrollFrameId = null
  }
  isAutoScrolling.value = false
}

const toggleAutoScroll = () => (isAutoScrolling.value ? stopAutoScroll() : startAutoScroll())
const adjustScrollSpeed = (delta: number) => (scrollSpeed.value = Math.max(1, Math.min(50, scrollSpeed.value + delta)))

function getOrCreatePageAnnotation(pageNumber: number): PageAnnotation {
  let page = pageAnnotations.get(pageNumber)
  if (!page) {
    page = { pageNumber, width: 1, height: 1, strokes: [] }
    pageAnnotations.set(pageNumber, page)
  }
  return page
}

function getNormalizedPoint(e: PointerEvent, canvas: HTMLCanvasElement): AnnotationPoint {
  const rect = canvas.getBoundingClientRect()
  return {
    x: Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width)),
    y: Math.max(0, Math.min(1, (e.clientY - rect.top) / rect.height)),
    p: e.pressure || 0.5
  }
}

function scheduleAnnotationSave(pageNumber: number) {
  const old = saveTimers.get(pageNumber)
  if (old) clearTimeout(old)
  const timer = window.setTimeout(async () => {
    saveTimers.delete(pageNumber)
    const page = getOrCreatePageAnnotation(pageNumber)
    await TabService.saveTabAnnotations(props.tabId, pageNumber, JSON.stringify(page.strokes))
  }, 300)
  saveTimers.set(pageNumber, timer)
}

async function ensurePageAnnotationLoaded(pageNumber: number) {
  if (loadedAnnotationPages.has(pageNumber)) return
  loadedAnnotationPages.add(pageNumber)
  try {
    const raw = await TabService.getTabAnnotations(props.tabId, pageNumber)
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      const page = getOrCreatePageAnnotation(pageNumber)
      page.strokes = parsed.filter((s: any) => s && Array.isArray(s.points) && s.points.length > 0)
    }
  } catch (e) {
    console.error('Failed to load annotations:', e)
  }
  renderPageAnnotations(pageNumber)
}
function drawStroke(ctx: CanvasRenderingContext2D, stroke: AnnotationStroke, canvas: HTMLCanvasElement) {
  if (stroke.points.length === 0) return
  const width = Math.max(1, canvas.clientWidth)
  const height = Math.max(1, canvas.clientHeight)
  const lineWidthPx = Math.max(1, stroke.lineWidth * width)
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.strokeStyle = stroke.color
  ctx.fillStyle = stroke.color
  if (stroke.type === 'highlight') {
    ctx.globalCompositeOperation = 'multiply'
    ctx.globalAlpha = 0.35
  } else {
    ctx.globalCompositeOperation = 'source-over'
    ctx.globalAlpha = 1
  }

  if (stroke.points.length === 1) {
    const p = stroke.points[0]
    ctx.beginPath()
    ctx.arc(p.x * width, p.y * height, lineWidthPx / 2, 0, Math.PI * 2)
    ctx.fill()
    return
  }

  ctx.beginPath()
  const first = stroke.points[0]
  ctx.moveTo(first.x * width, first.y * height)
  for (let i = 1; i < stroke.points.length; i++) {
    const p = stroke.points[i]
    ctx.lineWidth = lineWidthPx * (0.5 + (p.p ?? 0.5) * 0.5)
    ctx.lineTo(p.x * width, p.y * height)
  }
  ctx.stroke()
}

function renderPageAnnotations(pageNumber: number) {
  const canvas = pageCanvases.get(pageNumber)
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const page = getOrCreatePageAnnotation(pageNumber)
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  for (const stroke of page.strokes) drawStroke(ctx, stroke, canvas)
  if (activeStrokePageNumber === pageNumber && activeStroke) drawStroke(ctx, activeStroke, canvas)
  ctx.globalCompositeOperation = 'source-over'
  ctx.globalAlpha = 1
}

function resizeAnnotationCanvas(pageNumber: number) {
  const canvas = pageCanvases.get(pageNumber)
  if (!canvas) return
  const pageDiv = canvas.parentElement as HTMLDivElement | null
  if (!pageDiv) return

  const width = pageDiv.clientWidth
  const height = pageDiv.clientHeight
  const dpr = window.devicePixelRatio || 1
  if (width <= 0 || height <= 0) return

  const targetWidth = Math.floor(width * dpr)
  const targetHeight = Math.floor(height * dpr)
  if (canvas.width !== targetWidth || canvas.height !== targetHeight) {
    canvas.width = targetWidth
    canvas.height = targetHeight
    canvas.getContext('2d')?.setTransform(dpr, 0, 0, dpr, 0, 0)
  }
  canvas.style.width = `${width}px`
  canvas.style.height = `${height}px`
  const page = getOrCreatePageAnnotation(pageNumber)
  page.width = width
  page.height = height
  renderPageAnnotations(pageNumber)
}

function updateAnnotationCanvasInteractivity() {
  const interactive = annotationEnabled.value
  for (const canvas of pageCanvases.values()) {
    canvas.style.pointerEvents = interactive ? 'auto' : 'none'
    canvas.style.cursor = interactive ? (annotationTool.value === 'eraser' ? 'crosshair' : 'crosshair') : 'default'
  }
}

function eraseStrokeAtPoint(pageNumber: number, point: AnnotationPoint): boolean {
  const page = getOrCreatePageAnnotation(pageNumber)
  const canvas = pageCanvases.get(pageNumber)
  if (!canvas) return false

  const paddingX = Math.max(annotationLineWidth.value * 2, 12 / Math.max(canvas.clientWidth, 1))
  const paddingY = Math.max(annotationLineWidth.value * 2, 12 / Math.max(canvas.clientHeight, 1))
  const oldLength = page.strokes.length

  page.strokes = page.strokes.filter((stroke) => {
    let minX = 1, minY = 1, maxX = 0, maxY = 0
    for (const p of stroke.points) {
      minX = Math.min(minX, p.x)
      minY = Math.min(minY, p.y)
      maxX = Math.max(maxX, p.x)
      maxY = Math.max(maxY, p.y)
    }
    const hit = point.x >= minX - paddingX && point.x <= maxX + paddingX && point.y >= minY - paddingY && point.y <= maxY + paddingY
    return !hit
  })

  return oldLength !== page.strokes.length
}

function onAnnotationPointerDown(pageNumber: number, e: PointerEvent) {
  if (!annotationEnabled.value) return
  const canvas = pageCanvases.get(pageNumber)
  if (!canvas) return
  e.preventDefault()
  canvas.setPointerCapture(e.pointerId)

  if (annotationTool.value === 'eraser') {
    const point = getNormalizedPoint(e, canvas)
    if (eraseStrokeAtPoint(pageNumber, point)) {
      renderPageAnnotations(pageNumber)
      scheduleAnnotationSave(pageNumber)
    }
    return
  }

  isAnnotationDrawing = true
  activeStrokePageNumber = pageNumber
  activeStroke = {
    id: generateStrokeID(),
    type: annotationTool.value === 'highlight' ? 'highlight' : 'pen',
    color: annotationColor.value,
    lineWidth: annotationLineWidth.value,
    points: [getNormalizedPoint(e, canvas)]
  }
  renderPageAnnotations(pageNumber)
}

function onAnnotationPointerMove(pageNumber: number, e: PointerEvent) {
  if (!annotationEnabled.value) return
  const canvas = pageCanvases.get(pageNumber)
  if (!canvas) return

  if (annotationTool.value === 'eraser' && (e.buttons & 1) === 1) {
    const point = getNormalizedPoint(e, canvas)
    if (eraseStrokeAtPoint(pageNumber, point)) {
      renderPageAnnotations(pageNumber)
      scheduleAnnotationSave(pageNumber)
    }
    return
  }

  if (!isAnnotationDrawing || !activeStroke || activeStrokePageNumber !== pageNumber) return
  e.preventDefault()
  activeStroke.points.push(getNormalizedPoint(e, canvas))
  renderPageAnnotations(pageNumber)
}

function onAnnotationPointerUp(pageNumber: number, e: PointerEvent) {
  const canvas = pageCanvases.get(pageNumber)
  if (canvas?.hasPointerCapture(e.pointerId)) canvas.releasePointerCapture(e.pointerId)
  if (!isAnnotationDrawing || !activeStroke || activeStrokePageNumber !== pageNumber) return

  const page = getOrCreatePageAnnotation(pageNumber)
  if (activeStroke.points.length > 0) {
    page.strokes.push(activeStroke)
    scheduleAnnotationSave(pageNumber)
  }
  isAnnotationDrawing = false
  activeStroke = null
  activeStrokePageNumber = null
  renderPageAnnotations(pageNumber)
}
function getCurrentVisiblePageNumber(doc: Document): number {
  const selected = doc.querySelector('.page[data-page-number].selected') as HTMLDivElement | null
  if (selected) return parseInt(selected.dataset.pageNumber || '1', 10)
  const first = doc.querySelector('.page[data-page-number]') as HTMLDivElement | null
  return parseInt(first?.dataset.pageNumber || '1', 10)
}

function undoAnnotation() {
  const doc = getIframeDoc()
  if (!doc) return
  const pageNumber = getCurrentVisiblePageNumber(doc)
  const page = getOrCreatePageAnnotation(pageNumber)
  if (page.strokes.length === 0) return
  page.strokes.pop()
  renderPageAnnotations(pageNumber)
  scheduleAnnotationSave(pageNumber)
}

function clearCurrentPageAnnotations() {
  const doc = getIframeDoc()
  if (!doc) return
  const pageNumber = getCurrentVisiblePageNumber(doc)
  const page = getOrCreatePageAnnotation(pageNumber)
  if (page.strokes.length === 0) return
  page.strokes = []
  renderPageAnnotations(pageNumber)
  scheduleAnnotationSave(pageNumber)
}

function attachAnnotationCanvasToPage(pageDiv: HTMLDivElement) {
  const pageNumber = parseInt(pageDiv.dataset.pageNumber || '0', 10)
  if (!pageNumber || Number.isNaN(pageNumber)) return

  let canvas = pageCanvases.get(pageNumber)
  if (!canvas) {
    canvas = document.createElement('canvas')
    canvas.className = 'haya-annotation-layer'
    canvas.addEventListener('pointerdown', (e) => onAnnotationPointerDown(pageNumber, e))
    canvas.addEventListener('pointermove', (e) => onAnnotationPointerMove(pageNumber, e))
    canvas.addEventListener('pointerup', (e) => onAnnotationPointerUp(pageNumber, e))
    canvas.addEventListener('pointercancel', (e) => onAnnotationPointerUp(pageNumber, e))
    canvas.addEventListener('pointerleave', (e) => onAnnotationPointerUp(pageNumber, e))
    pageDiv.appendChild(canvas)
    pageCanvases.set(pageNumber, canvas)
    ensurePageAnnotationLoaded(pageNumber)
  } else if (canvas.parentElement !== pageDiv) {
    pageDiv.appendChild(canvas)
  }

  resizeAnnotationCanvas(pageNumber)
  updateAnnotationCanvasInteractivity()
}

function refreshAnnotationCanvases() {
  const doc = getIframeDoc()
  if (!doc) return
  doc.querySelectorAll('.page[data-page-number]').forEach((page) => attachAnnotationCanvasToPage(page as HTMLDivElement))
  for (const pageNumber of pageCanvases.keys()) resizeAnnotationCanvas(pageNumber)
}

function hideNativePdfAnnotationButtons(doc: Document) {
  for (const id of ['editorModeButtons', 'editorInk', 'editorFreeText', 'editorStamp']) {
    const el = doc.getElementById(id)
    if (el) (el as HTMLElement).style.display = 'none'
  }
}

function updateAnnotationToolbarUI() {
  const doc = getIframeDoc()
  if (!doc) return
  doc.getElementById('haya-annotation-toggle-btn')?.classList.toggle('toggled', annotationEnabled.value)
  doc.getElementById('haya-annotation-pen-btn')?.classList.toggle('toggled', annotationTool.value === 'pen')
  doc.getElementById('haya-annotation-highlight-btn')?.classList.toggle('toggled', annotationTool.value === 'highlight')
  doc.getElementById('haya-annotation-eraser-btn')?.classList.toggle('toggled', annotationTool.value === 'eraser')

  const colorInput = doc.getElementById('haya-annotation-color') as HTMLInputElement | null
  if (colorInput && colorInput.value !== annotationColor.value) colorInput.value = annotationColor.value

  const widthInput = doc.getElementById('haya-annotation-width') as HTMLInputElement | null
  if (widthInput) {
    const widthPercent = Math.round(annotationLineWidth.value * 1000) / 10
    if (widthInput.value !== String(widthPercent)) widthInput.value = String(widthPercent)
  }
}

function injectAnnotationToolbar(doc: Document) {
  const toolbarRight = doc.getElementById('toolbarViewerRight')
  if (!toolbarRight || doc.getElementById('haya-annotation-container')) return

  const button = (id: string, title: string, icon: string, onClick: () => void) => {
    const btn = doc.createElement('button')
    btn.id = id
    btn.className = 'haya-btn'
    btn.title = title
    btn.innerHTML = icon
    btn.onclick = onClick
    return btn
  }

  const container = doc.createElement('div')
  container.id = 'haya-annotation-container'
  container.className = 'haya-group'
  container.appendChild(button('haya-annotation-toggle-btn', 'Toggle annotation', ANNO_SVG, () => (annotationEnabled.value = !annotationEnabled.value)))
  container.appendChild(button('haya-annotation-pen-btn', 'Pen', PEN_SVG, () => { annotationTool.value = 'pen'; annotationEnabled.value = true }))
  container.appendChild(button('haya-annotation-highlight-btn', 'Highlighter', HIGHLIGHT_SVG, () => { annotationTool.value = 'highlight'; annotationEnabled.value = true }))
  container.appendChild(button('haya-annotation-eraser-btn', 'Eraser', ERASER_SVG, () => { annotationTool.value = 'eraser'; annotationEnabled.value = true }))
  container.appendChild(button('haya-annotation-undo-btn', 'Undo page', UNDO_SVG, () => undoAnnotation()))
  container.appendChild(button('haya-annotation-clear-btn', 'Clear page', CLEAR_SVG, () => clearCurrentPageAnnotations()))

  const colorInput = doc.createElement('input')
  colorInput.id = 'haya-annotation-color'
  colorInput.className = 'haya-color'
  colorInput.type = 'color'
  colorInput.value = annotationColor.value
  colorInput.onchange = () => (annotationColor.value = colorInput.value)
  container.appendChild(colorInput)

  const widthInput = doc.createElement('input')
  widthInput.id = 'haya-annotation-width'
  widthInput.className = 'haya-input'
  widthInput.type = 'number'
  widthInput.min = '0.1'
  widthInput.max = '2.0'
  widthInput.step = '0.1'
  widthInput.value = String(Math.round(annotationLineWidth.value * 1000) / 10)
  widthInput.onkeydown = (e: KeyboardEvent) => e.stopPropagation()
  widthInput.onchange = () => {
    const next = Math.max(0.1, Math.min(2, parseFloat(widthInput.value) || 0.5))
    annotationLineWidth.value = next / 100
    widthInput.value = String(next)
  }
  container.appendChild(widthInput)

  const wLabel = doc.createElement('span')
  wLabel.className = 'haya-label'
  wLabel.textContent = '%'
  container.appendChild(wLabel)

  toolbarRight.insertBefore(container, toolbarRight.firstChild)
  updateAnnotationToolbarUI()
}

function initAnnotationLayer(doc: Document) {
  hideNativePdfAnnotationButtons(doc)
  injectAnnotationToolbar(doc)
  refreshAnnotationCanvases()

  const viewer = doc.getElementById('viewer')
  if (!viewer) return

  pageMutationObserver?.disconnect()
  pageMutationObserver = new MutationObserver(() => refreshAnnotationCanvases())
  pageMutationObserver.observe(viewer, { childList: true, subtree: true, attributes: true, attributeFilter: ['style', 'class'] })

  viewerResizeObserver?.disconnect()
  viewerResizeObserver = new ResizeObserver(() => refreshAnnotationCanvases())
  viewerResizeObserver.observe(viewer)
}

function teardownAnnotationLayer() {
  pageMutationObserver?.disconnect()
  pageMutationObserver = null
  viewerResizeObserver?.disconnect()
  viewerResizeObserver = null
  for (const timer of saveTimers.values()) clearTimeout(timer)
  saveTimers.clear()
  pageCanvases.clear()
  pageAnnotations.clear()
  loadedAnnotationPages.clear()
}
function onIframeLoad() {
  const doc = getIframeDoc()
  if (!doc) return
  try {
    const isLight = document.body.getAttribute('data-theme') === 'light'
    doc.body.classList.add(isLight ? 'theme-light' : 'theme-dark')
    doc.addEventListener('keydown', handleKeydown)

    if (!doc.getElementById('haya-custom-style')) {
      const style = doc.createElement('style')
      style.id = 'haya-custom-style'
      style.textContent = `
        .haya-group { display: flex; align-items: center; gap: 4px; padding: 0 4px; margin-inline-end: 4px; border-inline-end: 1px solid var(--ht-border, #3e3e42); }
        .haya-btn { display: flex; align-items: center; justify-content: center; width: 28px; height: 28px; border: none; border-radius: 2px; background: transparent; color: var(--ht-text, #fff); cursor: pointer; padding: 0; }
        .haya-btn:hover { background: var(--ht-hover, #3e3e42); color: var(--ht-text, #fff); }
        .haya-btn.toggled { background: var(--ht-primary, #965233); color: #fff; }
        .haya-input { width: 48px; height: 24px; border: 1px solid var(--ht-border, #3e3e42); border-radius: 2px; background: var(--ht-bg, #1e1e1e); color: var(--ht-text, #fff); text-align: center; font-size: 12px; padding: 0 2px; -moz-appearance: textfield; }
        .haya-input::-webkit-inner-spin-button, .haya-input::-webkit-outer-spin-button { -webkit-appearance: none; margin: 0; }
        .haya-input:focus { outline: none; border-color: var(--ht-primary, #965233); }
        .haya-color { width: 28px; height: 24px; padding: 0; border: 1px solid var(--ht-border, #3e3e42); background: transparent; border-radius: 2px; cursor: pointer; }
        .haya-label { font-size: 11px; color: var(--ht-text-muted, #aaa); user-select: none; }
        .page { position: relative; }
        .haya-annotation-layer { position: absolute; inset: 0; width: 100%; height: 100%; z-index: 30; touch-action: none; pointer-events: none; background: transparent; }
      `
      doc.head.appendChild(style)
    }

    const toolbarRight = doc.getElementById('toolbarViewerRight')
    if (!toolbarRight) return

    if (!doc.getElementById('haya-metronome-container')) {
      const c = doc.createElement('div')
      c.id = 'haya-metronome-container'
      c.className = 'haya-group'

      const btn = doc.createElement('button')
      btn.id = 'haya-metronome-btn'
      btn.className = 'haya-btn'
      btn.title = 'Toggle Metronome (M)'
      btn.innerHTML = METRONOME_SVG
      btn.onclick = () => toggleMetronome()

      const input = doc.createElement('input')
      input.id = 'haya-metronome-bpm'
      input.className = 'haya-input'
      input.type = 'number'
      input.min = '20'
      input.max = '300'
      input.value = String(bpm.value)
      input.onkeydown = (e: KeyboardEvent) => e.stopPropagation()
      input.onchange = () => {
        const v = Math.max(20, Math.min(300, parseInt(input.value) || 120))
        bpm.value = v
        input.value = String(v)
      }

      const label = doc.createElement('span')
      label.className = 'haya-label'
      label.textContent = 'BPM'
      c.appendChild(btn); c.appendChild(input); c.appendChild(label)
      toolbarRight.insertBefore(c, toolbarRight.firstChild)
    }

    if (!doc.getElementById('haya-scroll-container')) {
      const c = doc.createElement('div')
      c.id = 'haya-scroll-container'
      c.className = 'haya-group'

      const btn = doc.createElement('button')
      btn.id = 'haya-scroll-btn'
      btn.className = 'haya-btn'
      btn.title = `Toggle Auto-Scroll (${settingsStore.settings.keyBindings.autoScroll.toUpperCase()})`
      btn.innerHTML = SCROLL_DOWN_SVG
      btn.onclick = () => toggleAutoScroll()

      const input = doc.createElement('input')
      input.id = 'haya-scroll-speed'
      input.className = 'haya-input'
      input.type = 'number'
      input.min = '1'
      input.max = '50'
      input.value = String(scrollSpeed.value)
      input.onkeydown = (e: KeyboardEvent) => e.stopPropagation()
      input.onchange = () => {
        const v = Math.max(1, Math.min(50, parseInt(input.value) || 1))
        scrollSpeed.value = v
        input.value = String(v)
      }

      const label = doc.createElement('span')
      label.className = 'haya-label'
      label.textContent = 'SPD'
      c.appendChild(btn); c.appendChild(input); c.appendChild(label)
      toolbarRight.insertBefore(c, toolbarRight.firstChild)
    }

    initAnnotationLayer(doc)
  } catch {
    // ignore
  }
}

onMounted(async () => {
  if (!isPdf.value || !tab.value) return
  await loadPdf()
  window.addEventListener('midi-scroll-down', handleMidiScrollDown)
  window.addEventListener('midi-scroll-up', handleMidiScrollUp)
  window.addEventListener('midi-play-pause', handleMidiPlayPause)
  window.addEventListener('midi-expression-scroll', handleMidiExpressionScroll as EventListener)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('midi-scroll-down', handleMidiScrollDown)
  window.removeEventListener('midi-scroll-up', handleMidiScrollUp)
  window.removeEventListener('midi-play-pause', handleMidiPlayPause)
  window.removeEventListener('midi-expression-scroll', handleMidiExpressionScroll as EventListener)
  stopMetronome()
  stopAutoScroll()
  teardownAnnotationLayer()
  if (audioCtx) {
    audioCtx.close()
    audioCtx = null
  }
  if (blobUrl.value) URL.revokeObjectURL(blobUrl.value)
})

const handleMidiScrollDown = () => props.visible && scrollPdf(300)
const handleMidiScrollUp = () => props.visible && scrollPdf(-300)
const handleMidiPlayPause = () => props.visible && toggleMetronome()

function handleMidiExpressionScroll(e: CustomEvent<number>) {
  if (!props.visible) return
  const viewerContainer = getIframeDoc()?.getElementById('viewerContainer')
  if (!viewerContainer) return
  const maxScroll = viewerContainer.scrollHeight - viewerContainer.clientHeight
  viewerContainer.scrollTop = maxScroll * e.detail
}

async function loadPdf() {
  if (!tab.value) return
  try {
    const port = await SettingsService.getFileServerPort()
    const url = `http://127.0.0.1:${port}/api/file/${props.tabId}`
    const pdfTheme = document.body.getAttribute('data-theme') === 'light' ? 1 : 2
    const appLang = settingsStore.settings.language || 'en'
    const localeMap: Record<string, string> = { en: 'en-US', 'zh-CN': 'zh-CN', 'zh-TW': 'zh-TW', ja: 'ja' }
    const pdfLocale = localeMap[appLang] || 'en-US'
    viewerUrl.value = `pdfjs/web/viewer.html?file=${encodeURIComponent(url)}#locale=${pdfLocale}&viewerCssTheme=${pdfTheme}`
  } catch (e) {
    console.error('Failed to load PDF:', e)
  }
}

function scrollPdf(amount: number) {
  const viewerContainer = getIframeDoc()?.getElementById('viewerContainer')
  if (viewerContainer) viewerContainer.scrollTop += amount
}
function handleKeydown(e: KeyboardEvent) {
  if (!props.visible) return
  const target = e.target as HTMLElement
  if (['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) || target.isContentEditable) return

  const keys = settingsStore.settings.keyBindings
  const key = e.key.toLowerCase()
  if (key === keys.scrollDown) {
    scrollPdf(100)
  } else if (key === keys.scrollUp) {
    scrollPdf(-100)
  } else if (key === keys.metronome) {
    toggleMetronome()
  } else if (key === keys.bpmPlus) {
    bpm.value = Math.min(300, bpm.value + 10)
  } else if (key === keys.bpmMinus) {
    bpm.value = Math.max(20, bpm.value - 10)
  } else if (key === keys.autoScroll) {
    toggleAutoScroll()
  } else if (e.key === keys.scrollSpeedUp) {
    adjustScrollSpeed(1)
  } else if (e.key === keys.scrollSpeedDown) {
    adjustScrollSpeed(-1)
  }
}

watch(() => props.visible, (newVal) => {
  if (newVal) {
    window.addEventListener('keydown', handleKeydown)
    refreshAnnotationCanvases()
  } else {
    window.removeEventListener('keydown', handleKeydown)
    stopAutoScroll()
  }
  if (newVal && iframeRef.value) window.dispatchEvent(new Event('resize'))
}, { immediate: true })

watch(isPlaying, (playing) => {
  const btn = getIframeDoc()?.getElementById('haya-metronome-btn')
  if (!btn) return
  btn.innerHTML = playing ? STOP_SVG : METRONOME_SVG
  btn.classList.toggle('toggled', playing)
})

watch(bpm, (val) => {
  const input = getIframeDoc()?.getElementById('haya-metronome-bpm') as HTMLInputElement | null
  if (input && input.value !== String(val)) input.value = String(val)
})

watch(isAutoScrolling, (scrolling) => {
  const btn = getIframeDoc()?.getElementById('haya-scroll-btn')
  if (!btn) return
  btn.innerHTML = scrolling ? SCROLL_PAUSE_SVG : SCROLL_DOWN_SVG
  btn.classList.toggle('toggled', scrolling)
})

watch(annotationEnabled, () => {
  updateAnnotationCanvasInteractivity()
  updateAnnotationToolbarUI()
})
watch(annotationTool, () => {
  updateAnnotationCanvasInteractivity()
  updateAnnotationToolbarUI()
})
watch(annotationColor, () => updateAnnotationToolbarUI())
watch(annotationLineWidth, () => updateAnnotationToolbarUI())
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
