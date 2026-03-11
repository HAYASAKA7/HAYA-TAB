/**
 * Base Wails API access layer.
 * This provides a centralized way to access the backend Go methods.
 */

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
    throw error;
  }
}
