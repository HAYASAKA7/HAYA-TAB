<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useContextMenu } from '@/composables/useContextMenu'
import MenuIcon from './MenuIcon.vue'

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
        <template v-if="item.type !== 'separator'">
          <MenuIcon v-if="item.icon" :name="item.icon" />
          <span class="menu-label">{{ item.label }}</span>
        </template>
      </li>
    </ul>
  </div>
</template>

<style scoped>
#context-menu-items li {
  display: flex;
  align-items: center;
  gap: 8px;
}

.menu-label {
  flex: 1;
}
</style>
