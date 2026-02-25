import { ref, shallowRef, toRaw, nextTick } from 'vue'

type TranslateFunction = (key: string, params?: Record<string, any>) => string

export function useAlphaTab(t?: TranslateFunction) {
  // Core API
  const api = shallowRef<any>(null)
  const isLoaded = ref(false)
  const isSoundFontLoaded = ref(false)
  const loadError = ref<string | null>(null)
  const isServerError = ref(false) // True if error is server-side (500, etc.), download won't help

  // Playback State
  const isPlaying = ref(false)
  const baseTempo = ref(120)
  const currentBpm = ref(120)
  const playbackSpeed = ref(1.0)
  const metronomeEnabled = ref(false)
  const isLooping = ref(false)

  // Track & Structure
  const tracks = ref<any[]>([])
  const selectedTrack = ref(0)
  const measureCount = ref(0)
  const markers = ref<Array<{ name: string; bar: number }>>([])

  // Selection State
  const selectionRange = ref<any>(null)
  const isSelectionActive = ref(false)
  const selectionHighlightStyles = ref<any[]>([])

  // --- Initialization ---

  function initialize(renderArea: HTMLElement, scrollElement: HTMLElement, options: any = {}) {
    if (!renderArea || !scrollElement) return

    // @ts-ignore - alphaTab is loaded globally
    api.value = new alphaTab.AlphaTabApi(renderArea, {
      core: {
        fontDirectory: '/alphatab/font/',
        useWorkers: false,
        ...options.core
      },
      player: {
        enablePlayer: true,
        enableCursor: true,
        soundFont: '/alphatab/soundfont/sonivox.sf3',
        scrollElement: scrollElement,
        ...options.player
      },
      display: {
        layoutMode: 'page',
        staveProfile: 'default',
        ...options.display
      }
    })

    setupListeners()
  }

  function setupListeners() {
    if (!api.value) return

    api.value.scoreLoaded.on((score: any) => {
      if (score && score.tempo) {
        baseTempo.value = score.tempo
        currentBpm.value = score.tempo
      }

      // Get Measure Count
      if (score && score.masterBars) {
        measureCount.value = score.masterBars.length

        // Extract markers
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
    })

    api.value.playerStateChanged.on((args: any) => {
      isPlaying.value = args.state === 1
    })

    api.value.playerReady.on(() => {
      isSoundFontLoaded.value = true
    })

    // playbackRangeHighlightChanged is manually handled by the consumer
    // via handleSelectionChange to allow for UI-specific logic (e.g. section mode protection)
  }

  function destroy() {
    if (api.value) {
      try {
        api.value.stop()
        api.value.destroy()
      } catch (e) {
        console.error('Error destroying alphaTab:', e)
      }
      api.value = null
    }
  }

  // --- Actions ---

  async function load(url: string, trackIndex?: number) {
    if (!api.value) return
    try {
        isLoaded.value = false
        loadError.value = null
        isServerError.value = false

        // Pre-check: fetch the URL to detect HTTP errors before AlphaTab tries to load
        const response = await fetch(url)
        if (!response.ok) {
          let errorMsg = t ? t('gpViewer.httpError', { status: response.status }) : `Failed to load file (HTTP ${response.status})`
          if (response.status === 403) {
            errorMsg = t ? t('gpViewer.accessDenied') : 'Access denied: The file may be too large or you may not have permission to access it'
            isServerError.value = true
          } else if (response.status === 404) {
            errorMsg = t ? t('gpViewer.fileNotFound') : 'File not found'
          } else if (response.status >= 500) {
            // Server errors - download won't help
            isServerError.value = true
            // Try to get more details from response
            const text = await response.text().catch(() => '')
            if (text.includes('403') || text.toLowerCase().includes('forbidden')) {
              errorMsg = t ? t('gpViewer.accessDeniedWebdav') : 'Access denied: The WebDAV server rejected the request (possibly file too large)'
            } else {
              errorMsg = t ? t('gpViewer.serverError') : 'Server error: Failed to stream file from cloud storage'
            }
          }
          throw new Error(errorMsg)
        }

        if (trackIndex !== undefined) {
             api.value.tracks = [trackIndex]
        }
        await api.value.load(url)
    } catch(e) {
        console.error("AlphaTab Load Failed", e)
        loadError.value = e instanceof Error ? e.message : String(e)
        throw e
    }
  }

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

  function playPause() {
    api.value?.playPause()
  }

  function stop() {
    api.value?.stop()
  }

  function toggleMetronome() {
    if (api.value) {
      metronomeEnabled.value = !metronomeEnabled.value
      api.value.metronomeVolume = metronomeEnabled.value ? 1 : 0
    }
  }

  function setBpm(bpm: number) {
    if (!api.value) return
    
    let val = bpm
    if (isNaN(val) || val < 20) val = 20
    if (val > 500) val = 500
    currentBpm.value = val

    const ratio = val / baseTempo.value
    api.value.playbackSpeed = ratio
    playbackSpeed.value = ratio
  }

  function setPlaybackSpeed(speed: number) {
    if (!api.value) return
    api.value.playbackSpeed = speed
    playbackSpeed.value = speed
    currentBpm.value = Math.round(baseTempo.value * speed)
  }

  function renderTrack(trackIndex: number) {
    if (!api.value || !api.value.score) return
    if (trackIndex >= 0 && api.value.score.tracks[trackIndex]) {
      api.value.renderTracks([api.value.score.tracks[trackIndex]])
      selectedTrack.value = trackIndex
    }
  }

  function toggleLoop() {
    if (!api.value) return

    isLooping.value = !isLooping.value
    if (isLooping.value && selectionRange.value) {
        api.value.playbackRange = toRaw(selectionRange.value)
        api.value.isLooping = true
        return true // Enabled
    } else {
        api.value.isLooping = false
        api.value.playbackRange = null
        isLooping.value = false
        return false // Disabled
    }
  }

  function clearSelection() {
    if (!api.value) return
    api.value.isLooping = false
    api.value.playbackRange = null
    selectionRange.value = null
    isSelectionActive.value = false
    isLooping.value = false
    selectionHighlightStyles.value = []
  }

  function playSelection() {
    if (!api.value || !selectionRange.value) return
    api.value.stop()
    api.value.playbackRange = toRaw(selectionRange.value)
    api.value.tickPosition = selectionRange.value.startTick
    nextTick(() => {
        if (api.value) {
            api.value.playPause()
        }
    })
  }

  // --- Logic ---

  function handleSelectionChange(args: any) {
    if (!args || !args.startBeat || !args.endBeat) {
        // We let the consumer handle "clearing" logic if needed, 
        // but here we just update state if valid clear.
        // Actually, GpViewer had specific logic about "sectionPlaybackMode" protecting clears.
        // We'll expose a method to force clear, but here we just reflect what AlphaTab says.
        // If AlphaTab says "cleared", we clear our state.
        // BUT, GpViewer's logic was: "if sectionPlaybackMode is on, ignore clears from AlphaTab"
        // This is UI logic. We should probably just pass the event data or update state, 
        // and let the component decide whether to ignore it?
        // Or better: The component calls this composable.
        // We will update state here. If component wants to protect it, it should manage that state?
        // No, AlphaTab sends the event.
        
        // Let's implement the standard logic here.
        // If the component needs to "protect" the selection, it might be tricky if we auto-clear here.
        // However, `playbackRangeHighlightChanged` usually fires when user interacts.
        
        selectionRange.value = null
        isSelectionActive.value = false
        selectionHighlightStyles.value = []
        return
    }

    const startBeat = args.startBeat
    const endBeat = args.endBeat
    const startTick = startBeat.absolutePlaybackStart
    const endTick = endBeat.absolutePlaybackStart + endBeat.playbackDuration

    if (startTick === endTick) {
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

    // Calculate Highlight Styles
    if (args.startBeatBounds && args.endBeatBounds) {
        const startBounds = args.startBeatBounds.visualBounds
        const endBounds = args.endBeatBounds.visualBounds
        
        if (startBounds && endBounds) {
            const styles: any[] = []
            const isSameLine = Math.abs(startBounds.y - endBounds.y) < 20
            
            if (isSameLine) {
                const minX = Math.min(startBounds.x, endBounds.x)
                const maxX = Math.max(startBounds.x + startBounds.w, endBounds.x + endBounds.w)
                styles.push({
                    left: (minX - 4) + 'px',
                    top: (startBounds.y - 4) + 'px',
                    width: (maxX - minX + 8) + 'px',
                    height: (startBounds.h + 8) + 'px'
                })
            } else {
                const boundsLookup = api.value?.boundsLookup || api.value?.renderer?.boundsLookup
                const groups = boundsLookup ? (boundsLookup.staveGroups || boundsLookup.staffSystems) : null
                
                if (groups) {
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
                                    
                                    if (beatEnd > startTick && beatStart < endTick) {
                                        const vb = beatBounds.visualBounds
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
                    
                    for (const group of lineGroups.values()) {
                        styles.push({
                            left: (group.minX - 4) + 'px',
                            top: (group.y - 4) + 'px',
                            width: (group.maxX - group.minX + 8) + 'px',
                            height: (group.h + 8) + 'px'
                        })
                    }
                } else {
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
  }

  return {
    api,
    isLoaded,
    isSoundFontLoaded,
    loadError,
    isServerError,
    isPlaying,
    baseTempo,
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
    clearSelection,
    playSelection,
    handleSelectionChange // Exported if component needs to manually call it or check it
  }
}
