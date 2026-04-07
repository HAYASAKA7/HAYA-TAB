<script setup lang="ts">
import { Events } from "@wailsio/runtime"
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const isVisible = ref(false)
const isCollapsed = ref(false)
const isHovered = ref(false)
const status = ref<'idle' | 'running' | 'success' | 'error'>('idle')
const message = ref('')
let collapseTimer: number | null = null
let dismissTimer: number | null = null

const startCollapseTimer = () => {
  if (collapseTimer) clearTimeout(collapseTimer)
  // 如果焦点在 toast 上，不启动自动折叠
  if (isHovered.value) return

  collapseTimer = window.setTimeout(() => {
    if (status.value === 'running') {
      isCollapsed.value = true
    }
  }, 2000)
}

const clearCollapseTimer = () => {
  if (collapseTimer) {
    clearTimeout(collapseTimer)
    collapseTimer = null
  }
}

const handleMouseEnter = () => {
  isHovered.value = true
  clearCollapseTimer()
}

const handleMouseLeave = () => {
  isHovered.value = false
  if (status.value === 'running' && !isCollapsed.value) {
    startCollapseTimer()
  }
}

watch([isVisible, isCollapsed], ([visible, collapsed]) => {
  if (!visible) {
    document.body.classList.remove('has-sync-toast')
    document.body.classList.remove('has-sync-toast-expanded')
  } else if (collapsed) {
    document.body.classList.add('has-sync-toast')
    document.body.classList.remove('has-sync-toast-expanded')
  } else {
    document.body.classList.add('has-sync-toast-expanded')
    document.body.classList.remove('has-sync-toast')
  }
})

const handleProgress = (ev: any) => {
  const data = ev.data ? (Array.isArray(ev.data) ? ev.data[0] : ev.data) : ev
  isVisible.value = true
  
  if (data.messageKey) {
    message.value = t(data.messageKey, data.errorArgs || {})
  } else {
    message.value = data.message || 'Processing...'
  }

  if (dismissTimer) {
    clearTimeout(dismissTimer)
    dismissTimer = null
  }

  if (data.status === 'start') {
    status.value = 'running'
    isCollapsed.value = false
    startCollapseTimer()
  } else if (data.status === 'progress') {
    status.value = 'running'
    // Do not alter collapse state here to avoid jitter if user just expanded it
  } else if (data.status === 'complete' || data.status === 'success') {
    status.value = 'success'
    isCollapsed.value = false // automatically unfold
    clearCollapseTimer()
    
    // Auto dismiss after 2s
    dismissTimer = window.setTimeout(() => {
      isVisible.value = false
      status.value = 'idle'
    }, 2000)
  } else if (data.status === 'error') {
    status.value = 'error'
    isCollapsed.value = false // automatically unfold
    clearCollapseTimer()
    // It stays visible until clicked
  }
}

let unregister: (() => void) | null = null

onMounted(() => {
  unregister = Events.On('webdav-sync-progress', handleProgress)
})

onUnmounted(() => {
  if (unregister) {
    unregister()
  }
  clearCollapseTimer()
  if (dismissTimer) clearTimeout(dismissTimer)

  document.body.classList.remove('has-sync-toast')
  document.body.classList.remove('has-sync-toast-expanded')
})

const closeOrExpand = () => {
  if (isCollapsed.value) {
    // Expand
    isCollapsed.value = false
    if (status.value === 'running') {
      startCollapseTimer()
    }
  } else {
    // Depending on status
    if (status.value === 'error' || status.value === 'success') {
      // Close
      isVisible.value = false
    } else {
      // Fold
      isCollapsed.value = true
    }
  }
}

const forceClose = (e: Event) => {
  e.stopPropagation()
  isVisible.value = false
  isCollapsed.value = false
}
</script>

<template>
  <div v-if="isVisible" id="sync-task-toast-container" 
       :class="{ collapsed: isCollapsed, 'status-error': status === 'error', 'status-success': status === 'success' }" 
       @click="closeOrExpand"
       @mouseenter="handleMouseEnter"
       @mouseleave="handleMouseLeave">
    
    <!-- Collapsed FAB mode -->
    <div v-if="isCollapsed" class="fab-mode" :title="message">
      <i v-if="status === 'running'" class="icon-sync rotate-icon"></i>
      <i v-else-if="status === 'success'" class="icon-checkbox"></i>
      <i v-else-if="status === 'error'" class="icon-close"></i>
    </div>
    
    <!-- Full Toast mode -->
    <div v-else class="toast-mode">
      <div class="toast-icon">
        <i v-if="status === 'running'" class="icon-sync rotate-icon"></i>
        <i v-else-if="status === 'success'" class="icon-checkbox"></i>
        <i v-else-if="status === 'error'" class="icon-close"></i>
      </div>
      <div class="toast-content">
        <div class="message">{{ message }}</div>
        <div class="sub-message" v-if="status === 'running'">{{ t('toast.syncTask.backgroundDesc') }}</div>
      </div>
      <div class="toast-actions" v-if="status === 'running'">
        <div class="toast-fold" @click.stop="isCollapsed = true" :title="t('toast.syncTask.fold')">
          <div class="fold-line"></div>
        </div>
      </div>
      <div class="toast-actions" v-else>
        <div class="toast-close" @click.stop="forceClose" :title="t('toast.syncTask.close')">
          <i class="icon-close"></i>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
#sync-task-toast-container {
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 10000;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  user-select: none;
}

.toast-mode {
  background: var(--card-bg, #2d2d2d);
  color: var(--text, #ffffff);
  border-radius: 8px;
  padding: 12px 16px;
  min-width: 240px;
  max-width: 320px;
  display: flex;
  align-items: center;
  gap: 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  border: 1px solid var(--border, #3e3e42);
  cursor: pointer;
  border-left: 4px solid var(--primary, #965233);
}

.status-error .toast-mode {
  border-left-color: var(--error-color, #e74c3c);
}

.status-success .toast-mode {
  border-left-color: var(--success-color, #27ae60);
}

.toast-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: var(--text, #ffffff);
}

.status-error .toast-icon {
  color: var(--error-color, #e74c3c);
}

.status-success .toast-icon {
  color: var(--success-color, #27ae60);
}

.toast-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.message {
  font-size: 14px;
  font-weight: bold;
  word-break: break-all;
}

.sub-message {
  font-size: 12px;
  color: var(--text-muted, #aaaaaa);
  margin-top: 2px;
}

.toast-actions {
  display: flex;
  align-items: center;
  justify-content: center;
}

.toast-fold {
  padding: 8px;
  cursor: pointer;
  opacity: 0.7;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.toast-fold:hover {
  opacity: 1;
}

.fold-line {
  width: 12px;
  height: 2px;
  background-color: var(--text, #ffffff);
  border-radius: 1px;
}

.toast-close {
  padding: 4px;
  border-radius: 4px;
  cursor: pointer;
  opacity: 0.7;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
}

.toast-close:hover {
  background: var(--hover, #3e3e42);
  opacity: 1;
}

.fab-mode {
  width: 48px;
  height: 48px;
  border-radius: 24px;
  background: var(--primary, #965233);
  opacity: 0.8;
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  cursor: pointer;
  color: #fff;
  transition: all 0.2s;
  font-size: 20px;
}

.fab-mode:hover {
  opacity: 1;
  transform: scale(1.05);
}

.status-error .fab-mode {
  background: var(--error-color, #e74c3c);
}

.status-success .fab-mode {
  background: var(--success-color, #27ae60);
}

.rotate-icon {
  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
