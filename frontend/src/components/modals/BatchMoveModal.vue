<script setup lang="ts">
import { ref, computed } from 'vue'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showToast } = useToast()

const selectedCategoryId = ref('')

const sortedCategories = computed(() => {
  return [...tabsStore.categories].sort((a, b) => {
    const pathA = tabsStore.getCategoryPath(a.id).join('/')
    const pathB = tabsStore.getCategoryPath(b.id).join('/')
    return pathA.localeCompare(pathB)
  })
})

async function handleSave() {
  try {
    const moved = await tabsStore.batchMoveTabs(selectedCategoryId.value)
    showToast(`Moved ${moved} tab(s)`)
    uiStore.hideBatchMoveModal()
  } catch (err) {
    showToast(String(err), 'error')
  }
}
</script>

<template>
  <div
    v-if="uiStore.batchMoveModalVisible"
    id="batch-move-modal"
    class="modal-overlay"
    @click.self="uiStore.hideBatchMoveModal"
  >
    <div class="modal">
      <h2>Move Selected Tabs</h2>

      <form @submit.prevent="handleSave">
        <div class="form-group">
          <label for="batch-move-select">Select Category</label>
          <select id="batch-move-select" v-model="selectedCategoryId">
            <option value="">(Root)</option>
            <option
              v-for="cat in sortedCategories"
              :key="cat.id"
              :value="cat.id"
            >
              {{ tabsStore.getCategoryPath(cat.id).join(' / ') }}
            </option>
          </select>
        </div>

        <div class="modal-actions">
          <button type="button" class="btn" @click="uiStore.hideBatchMoveModal">
            Cancel
          </button>
          <button type="submit" class="btn primary">
            Move
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
