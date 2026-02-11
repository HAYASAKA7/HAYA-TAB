// Tab represents a music tab file
export interface Tab {
  id: string
  title: string
  artist: string
  album: string
  filePath: string
  type: 'pdf' | 'gp' | 'unknown'
  isManaged: boolean
  coverPath: string
  categoryId: string
  country: string
  language: string
  tag: string
}

// Category represents a virtual folder for organizing tabs
export interface Category {
  id: string
  name: string
  parentId: string
}

// Settings represents application settings
export interface KeyBindings {
  scrollDown: string
  scrollUp: string
  metronome: string
  playPause: string
  stop: string
  bpmPlus: string
  bpmMinus: string
  toggleLoop: string
  clearSelection: string
  jumpToBar: string
}

export interface Settings {
  theme: 'dark' | 'light' | 'system'
  background: string
  bgType: 'url' | 'local' | ''
  openMethod: 'system' | 'inner'
  openGpMethod: 'system' | 'inner'
  audioDevice: string
  syncPaths: string[]
  syncStrategy: 'skip' | 'overwrite'
  autoSyncEnabled: boolean
  autoSyncFrequency: 'startup' | 'weekly' | 'monthly' | 'yearly'
  lastSyncTime: number
  keyBindings: KeyBindings
}

// TabsResponse represents a paginated response for tabs
export interface TabsResponse {
  tabs: Tab[]
  total: number
  page: number
  pageSize: number
  hasMore: boolean
}

// OpenedTab represents a tab that is currently open in a viewer
export interface OpenedTab {
  id: string
  tab: Tab
  isPinned: boolean
}

// ContextMenuItem represents an item in a context menu
export interface ContextMenuItem {
  label?: string
  action?: () => void
  type?: 'separator'
}

// ToastType represents the type of toast notification
export type ToastType = 'info' | 'error' | 'warning' | 'success'

// Toast represents a toast notification
export interface Toast {
  id: string
  message: string
  type: ToastType
}

// DragItem represents an item being dragged
export interface DragItem {
  type: 'tab' | 'category'
  id: string
}

// ViewType represents the current view
export type ViewType = 'home' | 'settings' | `pdf-${string}` | `gp-${string}`
