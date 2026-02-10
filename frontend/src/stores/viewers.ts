import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Tab } from '@/types'

export interface ViewerTab {
  id: string
  tab: Tab
  isPinned: boolean
}

export const useViewersStore = defineStore('viewers', () => {
  // State
  const openedTabs = ref<string[]>([])
  const pinnedTabs = ref<Set<string>>(new Set())
  const activeTabId = ref<string | null>(null)

  // Getters
  const sortedOpenedTabs = computed(() => {
    return [...openedTabs.value].sort((a, b) => {
      const aPinned = pinnedTabs.value.has(a)
      const bPinned = pinnedTabs.value.has(b)
      if (aPinned && !bPinned) return -1
      if (!aPinned && bPinned) return 1
      return 0
    })
  })

  // Actions
  function openTab(tab: Tab) {
    if (!openedTabs.value.includes(tab.id)) {
      openedTabs.value.push(tab.id)
    }
    activeTabId.value = tab.id
  }

  function closeTab(id: string) {
    const index = openedTabs.value.indexOf(id)
    if (index !== -1) {
      openedTabs.value.splice(index, 1)
    }
    pinnedTabs.value.delete(id)
    pinnedTabs.value = new Set(pinnedTabs.value)

    // If closing active tab, switch to another or home
    if (activeTabId.value === id) {
      if (openedTabs.value.length > 0) {
        activeTabId.value = openedTabs.value[openedTabs.value.length - 1]
      } else {
        activeTabId.value = null
      }
    }
  }

  function pinTab(id: string) {
    pinnedTabs.value.add(id)
    pinnedTabs.value = new Set(pinnedTabs.value)
  }

  function unpinTab(id: string) {
    pinnedTabs.value.delete(id)
    pinnedTabs.value = new Set(pinnedTabs.value)
  }

  function togglePin(id: string) {
    if (pinnedTabs.value.has(id)) {
      unpinTab(id)
    } else {
      pinTab(id)
    }
  }

  function isPinned(id: string) {
    return pinnedTabs.value.has(id)
  }

  function isOpen(id: string) {
    return openedTabs.value.includes(id)
  }

  function reorderTabs(fromIndex: number, toIndex: number) {
    const sorted = sortedOpenedTabs.value
    const tabId = sorted[fromIndex]
    const targetId = sorted[toIndex]

    // Find actual indices in openedTabs
    const actualFromIndex = openedTabs.value.indexOf(tabId)
    const actualToIndex = openedTabs.value.indexOf(targetId)

    if (actualFromIndex !== -1 && actualToIndex !== -1) {
      openedTabs.value.splice(actualFromIndex, 1)
      openedTabs.value.splice(actualToIndex, 0, tabId)
    }
  }

  function setActiveTab(id: string | null) {
    activeTabId.value = id
  }

  function closeAllTabs() {
    openedTabs.value = []
    pinnedTabs.value.clear()
    pinnedTabs.value = new Set()
    activeTabId.value = null
  }

  function closeUnpinnedTabs() {
    openedTabs.value = openedTabs.value.filter(id => pinnedTabs.value.has(id))
    if (activeTabId.value && !openedTabs.value.includes(activeTabId.value)) {
      activeTabId.value = openedTabs.value.length > 0 ? openedTabs.value[0] : null
    }
  }

  return {
    // State
    openedTabs,
    pinnedTabs,
    activeTabId,

    // Getters
    sortedOpenedTabs,

    // Actions
    openTab,
    closeTab,
    pinTab,
    unpinTab,
    togglePin,
    isPinned,
    isOpen,
    reorderTabs,
    setActiveTab,
    closeAllTabs,
    closeUnpinnedTabs
  }
})
