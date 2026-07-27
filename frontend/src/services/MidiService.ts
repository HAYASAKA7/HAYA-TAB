import { useSettingsStore } from '@/stores/settings'
import type { MidiMapping } from '@/types'

type MidiCallback = (message: WebMidi.MIDIMessageEvent) => void

export class MidiService {
  private static instance: MidiService
  private access: WebMidi.MIDIAccess | null = null
  private initialized = false
  private learnCallback: ((mapping: MidiMapping) => void) | null = null
  private eventListeners: Set<MidiCallback> = new Set()

  private constructor() {}

  public static getInstance(): MidiService {
    if (!MidiService.instance) {
      MidiService.instance = new MidiService()
    }
    return MidiService.instance
  }

  public async initialize() {
    if (this.initialized) return
    this.initialized = true

    if (!navigator.requestMIDIAccess) {
      console.warn('Web MIDI API not supported in this browser.')
      return
    }

    try {
      this.access = await navigator.requestMIDIAccess()
      this.access.onstatechange = () => this.updateInputs()
      this.updateInputs()
    } catch (err) {
      this.initialized = false
      console.error('Failed to access MIDI devices:', err)
    }
  }

  private updateInputs() {
    if (!this.access) return

    for (const input of this.access.inputs.values()) {
      input.onmidimessage = (event) => this.handleMidiMessage(event)
    }
  }

  private handleMidiMessage(event: WebMidi.MIDIMessageEvent) {
    const [status, data1, data2] = event.data
    const channel = status & 0x0f
    const typeNum = status & 0xf0

    let type: 'CC' | 'Note' | null = null
    if (typeNum === 0xb0) type = 'CC'
    else if (typeNum === 0x90 || typeNum === 0x80) type = 'Note'

    if (!type) return

    const mapping: MidiMapping = {
      type,
      number: data1,
      channel
    }

    // Handle MIDI Learn
    if (this.learnCallback) {
      // For CC, we usually want to trigger on value > 0
      // For Note, we only want Note On (0x90) with velocity > 0
      if (type === 'CC' || (typeNum === 0x90 && data2 > 0)) {
        this.learnCallback(mapping)
        this.learnCallback = null
        return
      }
    }

    // Global listeners
    this.eventListeners.forEach(cb => cb(event))

    // Handle Mapped Actions
    const settingsStore = useSettingsStore()
    if (!settingsStore.settings.midiSettings.enabled) return

    const { scrollDown, scrollUp, playPause, expressionScroll } = settingsStore.settings.midiSettings

    // Expression Pedal (Continuous)
    if (expressionScroll && this.isMatch(mapping, expressionScroll)) {
      if (mapping.type === 'CC') {
        const value = data2 / 127
        window.dispatchEvent(new CustomEvent('midi-expression-scroll', { detail: value }))
      }
      return
    }

    // Binary Actions (Pedal Switch)
    // We only trigger on "press" (value > 0 for CC, Velocity > 0 for Note On)
    const isPressed = type === 'CC' ? data2 > 0 : (typeNum === 0x90 && data2 > 0)
    if (!isPressed) return

    if (scrollDown && this.isMatch(mapping, scrollDown)) {
      window.dispatchEvent(new CustomEvent('midi-scroll-down'))
    } else if (scrollUp && this.isMatch(mapping, scrollUp)) {
      window.dispatchEvent(new CustomEvent('midi-scroll-up'))
    } else if (playPause && this.isMatch(mapping, playPause)) {
      window.dispatchEvent(new CustomEvent('midi-play-pause'))
    }
  }

  private isMatch(a: MidiMapping, b: MidiMapping): boolean {
    return a.type === b.type && a.number === b.number && a.channel === b.channel
  }

  public enterLearnMode(callback: (mapping: MidiMapping) => void) {
    this.learnCallback = callback
  }

  public cancelLearnMode() {
    this.learnCallback = null
  }

  public onMessage(callback: MidiCallback) {
    this.eventListeners.add(callback)
  }

  public offMessage(callback: MidiCallback) {
    this.eventListeners.delete(callback)
  }
}

export const midiService = MidiService.getInstance()
