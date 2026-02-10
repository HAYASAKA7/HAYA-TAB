import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ViewType } from '@/types'

export const useUIStore = defineStore('ui', () => {
  // State
  const currentView = ref<ViewType>('home')
  const sidebarCollapsed = ref(true)

  // Modal states
  const editModalVisible = ref(false)
  const categoryModalVisible = ref(false)
  const moveModalVisible = ref(false)
  const batchMoveModalVisible = ref(false)
  const confirmModalVisible = ref(false)
  const keyBindingsModalVisible = ref(false)

  // Modal data
  const editModalData = ref<any>(null)
  const categoryModalData = ref<any>(null)
  const moveModalTabId = ref('')
  const confirmModalData = ref<{
    title: string
    message: string
    okText: string
    isDanger: boolean
    onConfirm: () => void
  } | null>(null)

  // Context menu
  const contextMenuVisible = ref(false)
  const contextMenuX = ref(0)
  const contextMenuY = ref(0)
  const contextMenuItems = ref<any[]>([])

  // Actions
  function switchView(view: ViewType) {
    currentView.value = view
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function showEditModal(data: any) {
    editModalData.value = data
    editModalVisible.value = true
  }

  function hideEditModal() {
    editModalVisible.value = false
    editModalData.value = null
  }

  function showCategoryModal(data?: any) {
    categoryModalData.value = data || null
    categoryModalVisible.value = true
  }

  function hideCategoryModal() {
    categoryModalVisible.value = false
    categoryModalData.value = null
  }

  function showMoveModal(tabId: string) {
    moveModalTabId.value = tabId
    moveModalVisible.value = true
  }

  function hideMoveModal() {
    moveModalVisible.value = false
    moveModalTabId.value = ''
  }

  function showBatchMoveModal() {
    batchMoveModalVisible.value = true
  }

  function hideBatchMoveModal() {
    batchMoveModalVisible.value = false
  }

  function showConfirmModal(
    title: string,
    message: string,
    okText: string,
    isDanger: boolean,
    onConfirm: () => void
  ) {
    confirmModalData.value = { title, message, okText, isDanger, onConfirm }
    confirmModalVisible.value = true
  }

  function hideConfirmModal() {
    confirmModalVisible.value = false
    confirmModalData.value = null
  }

  function showKeyBindingsModal() {
    keyBindingsModalVisible.value = true
  }

  function hideKeyBindingsModal() {
    keyBindingsModalVisible.value = false
  }

  function showContextMenu(x: number, y: number, items: any[]) {
    contextMenuX.value = x
    contextMenuY.value = y
    contextMenuItems.value = items
    contextMenuVisible.value = true
  }

  function hideContextMenu() {
    contextMenuVisible.value = false
    contextMenuItems.value = []
  }

  return {
    // State
    currentView,
    sidebarCollapsed,
    editModalVisible,
    categoryModalVisible,
    moveModalVisible,
    batchMoveModalVisible,
    confirmModalVisible,
    keyBindingsModalVisible,
    editModalData,
    categoryModalData,
    moveModalTabId,
    confirmModalData,
    contextMenuVisible,
    contextMenuX,
    contextMenuY,
    contextMenuItems,

    // Actions
    switchView,
    toggleSidebar,
    showEditModal,
    hideEditModal,
    showCategoryModal,
    hideCategoryModal,
    showMoveModal,
    hideMoveModal,
    showBatchMoveModal,
    hideBatchMoveModal,
    showConfirmModal,
    hideConfirmModal,
    showKeyBindingsModal,
    hideKeyBindingsModal,
    showContextMenu,
    hideContextMenu
  }
})
