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
    await tabsStore.addTabToCategory(uiStore.moveModalTabId, selectedCategoryId.value)
    showToast('Added to category')
    uiStore.hideMoveModal()
  } catch (err) {
    showToast(String(err), 'error')
  }
}
</script>

<template>
  <div
    v-if="uiStore.moveModalVisible"
    id="move-modal"
    class="modal-overlay"
    @click.self="uiStore.hideMoveModal"
  >
    <div class="modal">
      <h2>Add to Category</h2>

      <form @submit.prevent="handleSave">
        <div class="form-group">
          <label for="move-select">Select Category</label>
          <select id="move-select" v-model="selectedCategoryId">
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
          <button type="button" class="btn" @click="uiStore.hideMoveModal">
            Cancel
          </button>
          <button type="submit" class="btn primary">
            Add
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
