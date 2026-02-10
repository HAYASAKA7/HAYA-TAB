<script setup lang="ts">
import { ref, computed } from 'vue'
import { useUIStore, useSettingsStore } from '@/stores'
import type { KeyBindings } from '@/types'

const uiStore = useUIStore()
const settingsStore = useSettingsStore()

const isOpen = computed(() => uiStore.keyBindingsModalVisible)

const editingKey = ref<string | null>(null)

const bindLabels: Record<keyof KeyBindings, string> = {
  scrollDown: 'Scroll Down',
  scrollUp: 'Scroll Up',
  metronome: 'Toggle Metronome',
  playPause: 'Play / Pause',
  stop: 'Stop / Rewind',
  bpmPlus: 'Increase BPM',
  bpmMinus: 'Decrease BPM'
}

function close() {
  uiStore.hideKeyBindingsModal()
  editingKey.value = null
}

function startEditing(key: string) {
  editingKey.value = key
}

function handleKeyDown(e: KeyboardEvent) {
  if (!editingKey.value) return

  e.preventDefault()
  e.stopPropagation()

  const newKey = e.key.toLowerCase() // Normalize to lowercase for simplicity
  
  // Validation: Check if key is already used? Maybe optional.
  // For now just allow binding.

  const keyField = editingKey.value as keyof KeyBindings
  settingsStore.settings.keyBindings[keyField] = newKey
  
  settingsStore.saveSettings()
  editingKey.value = null
}

function formatKey(key: string) {
  if (key === ' ') return 'Space'
  return key.toUpperCase()
}

</script>

<template>
  <div v-if="isOpen" id="key-binding-modal" class="modal-overlay" @click.self="close">
    <div class="modal" @keydown.stop>
      <h2>Key Bindings</h2>
      
      <div class="modal-body" tabindex="0" @keydown="handleKeyDown">
        <div v-if="editingKey" class="listening-overlay">
          <div class="listening-box">
            <p>Press a key for <strong>{{ bindLabels[editingKey as keyof KeyBindings] }}</strong></p>
            <button class="btn" @click.stop="editingKey = null">Cancel</button>
          </div>
        </div>

        <div class="bindings-list">
          <div 
            v-for="(label, field) in bindLabels" 
            :key="field" 
            class="binding-item"
          >
            <span class="binding-label">{{ label }}</span>
            <button 
              class="binding-key" 
              @click="startEditing(String(field))"
              title="Click to change"
            >
              {{ formatKey(settingsStore.settings.keyBindings[field]) }}
            </button>
          </div>
        </div>
      </div>
      
      <div class="modal-actions">
        <button class="btn primary" @click="close">Done</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal {
  width: 500px; /* Slightly wider than default */
  max-width: 90vw;
}

.modal-body {
  position: relative;
  min-height: 300px;
  outline: none; /* Make div focusable for keydown listener */
  margin-top: 20px;
}

.bindings-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.binding-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background-color: var(--bg);
  border: 1px solid var(--border);
  border-radius: 6px;
}

.binding-label {
  font-weight: 500;
  color: var(--text);
}

.binding-key {
  background-color: var(--card-bg);
  color: var(--primary);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 5px 15px;
  min-width: 80px;
  text-align: center;
  cursor: pointer;
  font-family: monospace;
  font-size: 1.1em;
  font-weight: bold;
  transition: all 0.2s;
  text-transform: uppercase;
}

.binding-key:hover {
  border-color: var(--primary);
  background: var(--hover);
}

.listening-overlay {
  position: absolute;
  top: -20px;
  left: -20px;
  right: -20px;
  bottom: -20px;
  background-color: rgba(0, 0, 0, 0.85); /* Darker backdrop within modal */
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 10;
  border-radius: 8px;
  backdrop-filter: blur(2px);
}

.listening-box {
  background-color: var(--card-bg);
  padding: 30px;
  border-radius: 8px;
  text-align: center;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
  border: 1px solid var(--primary);
}

.listening-box p {
  margin-bottom: 20px;
  font-size: 1.1em;
  color: var(--text);
}
</style>
