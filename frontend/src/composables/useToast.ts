import { ref } from 'vue'
import i18n from '@/i18n'
import type { Toast, ToastType } from '@/types'
import type { AppApiError } from '@/services/api'

const toasts = ref<Toast[]>([])
const MAX_TOASTS = 3
let toastId = 0

function getErrorMessage(error: unknown): string {
  if (error && typeof error === 'object') {
    const appError = error as Partial<AppApiError>
    if (typeof appError.i18nKey === 'string' && appError.i18nKey.startsWith('errors.')) {
      return i18n.global.t(appError.i18nKey, appError.i18nArgs ?? {})
    }
    if (typeof appError.message === 'string') {
      if (appError.message.startsWith('errors.')) {
        return i18n.global.t(appError.message)
      }
      return appError.message
    }
  }

  if (typeof error === 'string') {
    return error.startsWith('errors.') ? i18n.global.t(error) : error
  }

  return i18n.global.t('errors.unknown')
}

export function useToast() {
  function showToast(message: string, type: ToastType = 'info') {
    const id = `toast-${++toastId}`
    const toast: Toast = { id, message, type }

    // Remove oldest toasts if limit reached
    while (toasts.value.length >= MAX_TOASTS) {
      toasts.value.shift()
    }

    toasts.value.push(toast)

    // Auto remove after 3 seconds
    setTimeout(() => {
      removeToast(id)
    }, 3000)

    return id
  }

  function showErrorToast(error: unknown, prefix?: string) {
    const message = getErrorMessage(error)
    showToast(prefix ? `${prefix}: ${message}` : message, 'error')
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
    showErrorToast,
    removeToast
  }
}
