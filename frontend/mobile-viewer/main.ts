import * as alphaTab from '@coderline/alphatab'

type BridgeErrorCode =
  | 'unsupported_version'
  | 'unknown_command'
  | 'invalid_payload'
  | 'viewer_error'

type BridgeResult =
  | { ok: true }
  | { ok: false; error: { code: BridgeErrorCode } }

type BridgeCommand = {
  version?: unknown
  type?: unknown
  payload?: unknown
}

declare global {
  interface Window {
    hayaTabViewer: {
      receive(command: unknown): Promise<BridgeResult>
    }
    webkit?: {
      messageHandlers?: {
        hayaBridge?: {
          postMessage(message: unknown): Promise<unknown> | void
        }
      }
    }
  }
}

const scoreElement = document.querySelector<HTMLElement>('#score')
const statusElement = document.querySelector<HTMLElement>('#status')
let api: alphaTab.AlphaTabApi | null = null
let baseTempo = 120

function resultError(code: BridgeErrorCode): BridgeResult {
  return { ok: false, error: { code } }
}

function postToNative(message: unknown): void {
  try {
    const reply = window.webkit?.messageHandlers?.hayaBridge?.postMessage(message)
    if (reply instanceof Promise) {
      void reply.catch(() => undefined)
    }
  } catch {
    // The native side owns bridge diagnostics; no document data is logged here.
  }
}

function ensureAPI(): alphaTab.AlphaTabApi | null {
  if (api || !scoreElement) return api

  api = new alphaTab.AlphaTabApi(scoreElement, {
    core: {
      fontDirectory: new URL('./font/', window.location.href).href,
      useWorkers: false
    },
    player: {
      enablePlayer: true,
      enableCursor: true,
      soundFont: new URL('./soundfont/sonivox.sf3', window.location.href).href
    },
    display: {
      layoutMode: 'page',
      staveProfile: 'default'
    }
  })

  api.scoreLoaded.on(score => {
    baseTempo = score.tempo > 0 ? score.tempo : 120
    if (statusElement) statusElement.hidden = true
    postToNative({ version: 1, type: 'loaded' })
  })
  api.error.on(() => {
    if (statusElement) {
      statusElement.hidden = false
      statusElement.textContent = 'This score could not be opened.'
    }
    postToNative({
      version: 1,
      type: 'error',
      payload: { code: 'viewer_error' }
    })
  })
  return api
}

function decodeBase64(value: unknown): Uint8Array | null {
  if (typeof value !== 'string' || value.length === 0) return null
  try {
    const binary = atob(value)
    const bytes = new Uint8Array(binary.length)
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index)
    }
    return bytes
  } catch {
    return null
  }
}

async function receive(commandValue: unknown): Promise<BridgeResult> {
  if (!commandValue || typeof commandValue !== 'object') {
    return resultError('invalid_payload')
  }
  const command = commandValue as BridgeCommand
  if (command.version !== 1) {
    return resultError('unsupported_version')
  }
  if (typeof command.type !== 'string') {
    return resultError('invalid_payload')
  }

  switch (command.type) {
    case 'load': {
      const payload = command.payload as { base64?: unknown } | undefined
      const bytes = decodeBase64(payload?.base64)
      if (!bytes) return resultError('invalid_payload')
      try {
        ensureAPI()?.load(bytes)
        return { ok: true }
      } catch {
        return resultError('viewer_error')
      }
    }
    case 'playPause':
      try {
        api?.playPause()
        return { ok: true }
      } catch {
        return resultError('viewer_error')
      }
    case 'stop':
      try {
        api?.stop()
        return { ok: true }
      } catch {
        return resultError('viewer_error')
      }
    case 'setTempo': {
      const payload = command.payload as { beatsPerMinute?: unknown } | undefined
      const beatsPerMinute = payload?.beatsPerMinute
      if (
        typeof beatsPerMinute !== 'number'
        || !Number.isFinite(beatsPerMinute)
        || beatsPerMinute < 30
        || beatsPerMinute > 300
      ) {
        return resultError('invalid_payload')
      }
      if (api) api.playbackSpeed = beatsPerMinute / baseTempo
      return { ok: true }
    }
    default:
      return resultError('unknown_command')
  }
}

window.hayaTabViewer = { receive }
postToNative({ version: 1, type: 'ready' })
