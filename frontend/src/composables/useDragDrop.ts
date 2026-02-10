import { ref } from 'vue'
import type { DragItem } from '@/types'

const draggedItem = ref<DragItem | null>(null)
const draggedSidebarItem = ref<string | null>(null)

export function useDragDrop() {
  function startDrag(item: DragItem) {
    draggedItem.value = item
  }

  function endDrag() {
    draggedItem.value = null
  }

  function startSidebarDrag(tabId: string) {
    draggedSidebarItem.value = tabId
  }

  function endSidebarDrag() {
    draggedSidebarItem.value = null
  }

  function handleDragOver(e: DragEvent, element: HTMLElement) {
    e.preventDefault()
    if (!draggedItem.value) return
    element.classList.add('drag-over')
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = 'move'
    }
  }

  function handleDragLeave(_e: DragEvent, element: HTMLElement) {
    element.classList.remove('drag-over')
  }

  return {
    draggedItem,
    draggedSidebarItem,
    startDrag,
    endDrag,
    startSidebarDrag,
    endSidebarDrag,
    handleDragOver,
    handleDragLeave
  }
}
