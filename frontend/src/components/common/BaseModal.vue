<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useSlots, watch } from 'vue'

type ModalSize = 'small' | 'medium' | 'large'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  size?: ModalSize
  closeOnOverlay?: boolean
}>(), {
  size: 'small',
  closeOnOverlay: true,
})

const emit = defineEmits<{
  close: []
}>()

const slots = useSlots()
const dialogRef = ref<HTMLElement | null>(null)
const titleId = `modal-title-${Math.random().toString(36).slice(2)}`
let previouslyFocusedElement: HTMLElement | null = null

function close() {
  emit('close')
}

function handleOverlayClose() {
  if (props.closeOnOverlay) {
    close()
  }
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return

  event.preventDefault()
  event.stopPropagation()
  close()
}

function restoreFocus() {
  if (previouslyFocusedElement?.isConnected) {
    previouslyFocusedElement.focus({ preventScroll: true })
  }
  previouslyFocusedElement = null
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      previouslyFocusedElement = document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null
      document.addEventListener('keydown', handleKeydown, true)
      await nextTick()
      dialogRef.value?.focus({ preventScroll: true })
      return
    }

    document.removeEventListener('keydown', handleKeydown, true)
    restoreFocus()
  },
  { flush: 'post' },
)

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeydown, true)
  restoreFocus()
})
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="modal-overlay"
      data-testid="modal-overlay"
      @click.self="handleOverlayClose"
    >
      <section
        ref="dialogRef"
        class="modal-dialog"
        :class="`modal-dialog--${size}`"
        :data-modal-size="size"
        data-testid="base-modal"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        tabindex="-1"
      >
        <header class="modal-header">
          <h2 :id="titleId">{{ title }}</h2>
        </header>

        <div class="modal-body">
          <slot />
        </div>

        <footer v-if="slots.actions" class="modal-actions">
          <slot name="actions" />
        </footer>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  padding: 16px;
  background: rgba(0, 0, 0, 0.8);
}

.modal-dialog {
  --modal-width: 420px;

  box-sizing: border-box;
  width: min(var(--modal-width), calc(100vw - 32px));
  min-width: 0;
  max-height: calc(100dvh - 32px);
  display: flex;
  flex-direction: column;
  padding: 20px 24px;
  overflow: hidden;
  color: var(--text);
  background: var(--card-bg);
  border-radius: 8px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
  outline: none;
}

.modal-dialog--medium {
  --modal-width: 500px;
}

.modal-dialog--large {
  --modal-width: 800px;
  height: min(80dvh, calc(100dvh - 32px));
}

.modal-header {
  flex: none;
}

.modal-header h2 {
  margin: 0 0 16px;
  font-size: 1.1rem;
  font-weight: 600;
}

.modal-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.modal-actions {
  display: flex;
  flex: none;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

.modal-actions :slotted(.modal-actions__leading) {
  margin-right: auto;
}
</style>
