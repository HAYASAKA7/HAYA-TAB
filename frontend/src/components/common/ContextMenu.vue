<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useContextMenu } from '@/composables/useContextMenu'

const { visible, x, y, items, hide, handleItemClick } = useContextMenu()

function handleGlobalClick() {
  hide()
}

onMounted(() => {
  document.addEventListener('click', handleGlobalClick)
})

onUnmounted(() => {
  document.removeEventListener('click', handleGlobalClick)
})
</script>

<template>
  <div
    v-show="visible"
    id="context-menu"
    class="context-menu"
    :class="{ hidden: !visible }"
    :style="{ left: x + 'px', top: y + 'px' }"
    @click.stop
  >
    <ul id="context-menu-items">
      <li
        v-for="(item, index) in items"
        :key="index"
        :class="{ separator: item.type === 'separator' }"
        @click="item.type !== 'separator' && handleItemClick(item)"
      >
        {{ item.type !== 'separator' ? item.label : '' }}
      </li>
    </ul>
  </div>
</template>
