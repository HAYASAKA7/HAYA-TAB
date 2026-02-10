<script setup lang="ts">
import { ref, watch } from 'vue'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'

const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showToast } = useToast()

const categoryId = ref('')
const categoryName = ref('')

// Watch for modal data changes
watch(() => uiStore.categoryModalData, (data) => {
  if (data) {
    categoryId.value = data.id || ''
    categoryName.value = data.name || ''
  } else {
    categoryId.value = ''
    categoryName.value = ''
  }
}, { immediate: true })

async function handleSave() {
  if (!categoryName.value.trim()) return

  try {
    const existingCategory = tabsStore.categories.find(c => c.id === categoryId.value)

    await tabsStore.addCategory({
      id: categoryId.value,
      name: categoryName.value.trim(),
      parentId: categoryId.value
        ? existingCategory?.parentId || ''
        : tabsStore.currentCategoryId
    })

    uiStore.hideCategoryModal()
  } catch (err) {
    showToast(String(err), 'error')
  }
}
</script>

<template>
  <div
    v-if="uiStore.categoryModalVisible"
    id="category-modal"
    class="modal-overlay"
    @click.self="uiStore.hideCategoryModal"
  >
    <div class="modal">
      <h2>{{ categoryId ? 'Rename Category' : 'New Category' }}</h2>

      <form @submit.prevent="handleSave">
        <div class="form-group">
          <label for="cat-name">Name</label>
          <input
            id="cat-name"
            type="text"
            v-model="categoryName"
            required
            autofocus
          />
        </div>

        <div class="modal-actions">
          <button type="button" class="btn" @click="uiStore.hideCategoryModal">
            Cancel
          </button>
          <button type="submit" class="btn primary">
            Save
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
