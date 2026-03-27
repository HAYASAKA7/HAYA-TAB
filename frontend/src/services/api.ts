/**
 * Base Wails API access layer.
 * This provides a centralized way to access the backend Go methods.
 */

export interface AppApiError {
  code: string
  i18nKey: string
  i18nArgs?: Record<string, unknown>
  message: string
  technical?: string
}

function normalizeBackendError(error: unknown): AppApiError {
  if (error && typeof error === 'object') {
    const candidate = error as Record<string, unknown>

    if (typeof candidate.code === 'string' && typeof candidate.message === 'string') {
      return {
        code: candidate.code,
        i18nKey: typeof candidate.i18nKey === 'string' ? candidate.i18nKey : candidate.code,
        i18nArgs: typeof candidate.i18nArgs === 'object' && candidate.i18nArgs !== null ? candidate.i18nArgs as Record<string, unknown> : undefined,
        message: candidate.message,
        technical: typeof candidate.technical === 'string' ? candidate.technical : undefined,
      }
    }

    if ('error' in candidate && candidate.error && typeof candidate.error === 'object') {
      return normalizeBackendError(candidate.error)
    }
  }

  if (typeof error === 'string') {
    try {
      return normalizeBackendError(JSON.parse(error))
    } catch {
      if (error.startsWith('errors.')) {
        return {
          code: error,
          i18nKey: error,
          message: error,
        }
      }

      return {
        code: 'errors.unknown',
        i18nKey: 'errors.unknown',
        message: error,
        technical: error,
      }
    }
  }

  if (error instanceof Error) {
    return normalizeBackendError(error.message)
  }

  return {
    code: 'errors.unknown',
    i18nKey: 'errors.unknown',
    message: 'errors.unknown',
  }
}

/**
 * Executes a call to the Wails backend.
 * @param method The method name on the App struct
 * @param args Arguments for the method
 * @returns Promise with the result
 */
export async function callBackend<T = any>(method: string, ...args: any[]): Promise<T> {
  const w = window as any;

  // Wails can bind under different namespaces depending on how NewApp is initialized.
  // We check both 'main' and 'app' to be resilient.
  const app = (w.go?.main?.App) || (w.go?.app?.App);

  if (!app) {
    console.error(`Wails backend App not found in window.go.main or window.go.app. Cannot call ${method}`);
    throw new Error('Backend not initialized');
  }

  const func = app[method];
  if (typeof func !== 'function') {
    console.error(`Method ${method} not found on backend App. Registered methods:`, Object.keys(app));
    throw new Error(`Method ${method} not found: ${method}`);
  }

  try {
    return await func(...args);
  } catch (error) {
    console.error(`Error calling backend method ${method}:`, error);
    throw normalizeBackendError(error);
  }
}
