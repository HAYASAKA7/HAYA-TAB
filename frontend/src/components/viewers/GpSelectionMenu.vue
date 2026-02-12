<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'

const props = defineProps<{
  visible: boolean
  x: number
  y: number
  isLooping: boolean
  currentSpeed: number
  isPlaying?: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-loop'): void
  (e: 'set-speed', speed: number): void
  (e: 'play-selection'): void
  (e: 'close'): void
}>()

const speedOptions = [0.25, 0.5, 0.75, 1.0, 1.25, 1.5]
const menuRef = ref<HTMLElement | null>(null)
const clampedPos = ref({ left: 0, top: 0 })

const displaySpeed = computed(() => {
  return Math.round(props.currentSpeed * 100) + '%'
})

watch(() => props.visible, async (val) => {
  if (!val) return
  await nextTick()
  clampPosition()
})

watch(() => [props.x, props.y], () => {
  if (props.visible) clampPosition()
})

function clampPosition() {
  const el = menuRef.value
  if (!el) return

  const parent = el.offsetParent as HTMLElement | null
  if (!parent) return

  const menuW = el.offsetWidth
  const menuH = el.offsetHeight
  const parentW = parent.clientWidth
  const scrollLeft = parent.scrollLeft
  const scrollTop = parent.scrollTop

  // Default: centered above the point (matches transform: translate(-50%, -100%))
  let left = props.x - menuW / 2
  let top = props.y - menuH

  // Clamp horizontal: keep within visible scroll area
  const minLeft = scrollLeft + 8
  const maxLeft = scrollLeft + parentW - menuW - 8
  left = Math.max(minLeft, Math.min(left, maxLeft))

  // Clamp vertical: if menu would go above visible area, flip below
  const minTop = scrollTop + 8
  if (top < minTop) {
    top = props.y + 20
  }

  clampedPos.value = { left, top }
}
</script>

<template>
  <Transition name="menu-pop">
    <div
      v-if="visible"
      ref="menuRef"
      class="gp-selection-menu"
      :style="{ left: clampedPos.left + 'px', top: clampedPos.top + 'px' }"
      @mousedown.stop
      @click.stop
    >
      <div class="menu-content">
        <!-- Play Selection -->
        <button 
          class="menu-btn play-btn" 
          :class="{ active: isPlaying }"
          @click="emit('play-selection')"
          title="Play Selection"
        >
          <span :class="isPlaying ? 'icon-pause' : 'icon-play'"></span>
        </button>

        <!-- Loop Toggle -->
        <button 
          class="menu-btn" 
          :class="{ active: isLooping }" 
          @click="emit('toggle-loop')"
          title="Loop Selection (L)"
        >
          <span class="icon-loop"></span>
        </button>

        <div class="divider"></div>

        <!-- Speed Controls -->
        <div class="speed-section">
          <span class="speed-label">Speed</span>
          <div class="speed-controls">
            <button 
              v-for="speed in speedOptions" 
              :key="speed"
              class="speed-btn"
              :class="{ active: currentSpeed === speed }"
              @click="emit('set-speed', speed)"
              :title="`${speed * 100}% Speed`"
            >
              {{ speed === 1 ? '1×' : speed + '×' }}
            </button>
          </div>
          <span class="current-speed">{{ displaySpeed }}</span>
        </div>

        <div class="divider"></div>
        
        <!-- Close -->
        <button class="menu-btn close-btn" @click="emit('close')" title="Close (Esc)">
          <span class="icon-close"></span>
        </button>
      </div>
      <div class="menu-arrow"></div>
    </div>
  </Transition>
</template>

<style scoped>
.gp-selection-menu {
  position: absolute;
  z-index: 1000;
  padding-bottom: 12px; /* Space for arrow */
  filter: drop-shadow(0 8px 24px rgba(0,0,0,0.25));
}

/* Menu pop animation */
.menu-pop-enter-active {
  animation: menuPopIn 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.menu-pop-leave-active {
  animation: menuPopOut 0.15s ease-in;
}

@keyframes menuPopIn {
  0% {
    opacity: 0;
    transform: translateY(20%) scale(0.9);
  }
  100% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes menuPopOut {
  0% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
  100% {
    opacity: 0;
    transform: translateY(10%) scale(0.95);
  }
}

.menu-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 6px 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
}

.menu-arrow {
  position: absolute;
  bottom: 6px;
  left: 50%;
  transform: translateX(-50%) rotate(45deg);
  width: 14px;
  height: 14px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  border-bottom: 1px solid var(--border-color);
  z-index: 1001;
}

.divider {
  width: 1px;
  height: 28px;
  background: var(--border-color);
  margin: 0 4px;
}

.menu-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  width: 36px;
  height: 36px;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.menu-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
  transform: scale(1.05);
}

.menu-btn:active {
  transform: scale(0.95);
}

.menu-btn.active {
  color: var(--primary-color);
  background: var(--bg-tertiary);
}

.play-btn.active {
  background: var(--primary-color);
  color: white;
}

/* Speed Section */
.speed-section {
  display: flex;
  align-items: center;
  gap: 8px;
}

.speed-label {
  font-size: 0.7rem;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 500;
}

.speed-controls {
  display: flex;
  gap: 2px;
  background: var(--bg-tertiary);
  padding: 3px;
  border-radius: 6px;
}

.speed-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  padding: 4px 8px;
  font-size: 0.75rem;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
  font-weight: 500;
}

.speed-btn:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.speed-btn.active {
  background: var(--primary-color);
  color: white;
  box-shadow: 0 2px 6px rgba(150, 82, 51, 0.3);
}

.current-speed {
  font-size: 0.8rem;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
  min-width: 36px;
  text-align: right;
}

.close-btn:hover {
  color: var(--error-color);
  background: rgba(231, 76, 60, 0.1);
}

/* Icon sizes */
.menu-btn span {
  width: 1.2em;
  height: 1.2em;
}

.play-btn span {
  width: 1.3em;
  height: 1.3em;
}
</style>
