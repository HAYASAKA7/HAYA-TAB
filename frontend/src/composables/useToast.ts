import { ref } from 'vue'
import type { Toast, ToastType } from '@/types'

const toasts = ref<Toast[]>([])
let toastId = 0

export function useToast() {
  function showToast(message: string, type: ToastType = 'info') {
    const id = `toast-${++toastId}`
    const toast: Toast = { id, message, type }
    toasts.value.push(toast)

    // Auto remove after 3 seconds
    setTimeout(() => {
      removeToast(id)
    }, 3000)

    return id
  }

  function removeToast(id: string) {
    const index = toasts.value.findIndex(t => t.id === id)
    if (index !== -1) {
      toasts.value.splice(index, 1)
    }
  }

  return {
    toasts,
    showToast,
    removeToast
  }
}
