import { ref } from 'vue'
import type { ContextMenuItem } from '@/types'

const visible = ref(false)
const x = ref(0)
const y = ref(0)
const items = ref<ContextMenuItem[]>([])

export function useContextMenu() {
  function show(pageX: number, pageY: number, menuItems: ContextMenuItem[]) {
    // Adjust position if menu would go off screen
    const menuWidth = 150
    const menuHeight = menuItems.length * 35

    let adjustedX = pageX
    let adjustedY = pageY

    if (pageX + menuWidth > window.innerWidth) {
      adjustedX = pageX - menuWidth
    }
    if (pageY + menuHeight > window.innerHeight) {
      adjustedY = pageY - menuHeight
    }

    x.value = adjustedX
    y.value = adjustedY
    items.value = menuItems
    visible.value = true
  }

  function hide() {
    visible.value = false
    items.value = []
  }

  function handleItemClick(item: ContextMenuItem) {
    if (item.action) {
      item.action()
    }
    hide()
  }

  return {
    visible,
    x,
    y,
    items,
    show,
    hide,
    handleItemClick
  }
}
