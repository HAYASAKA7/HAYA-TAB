<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showToast } = useToast()

const categoryId = ref('')
const categoryName = ref('')
const coverPath = ref('')

// Watch for modal data changes
watch(() => uiStore.categoryModalData, (data) => {
  if (data) {
    categoryId.value = data.id || ''
    categoryName.value = data.name || ''
    coverPath.value = data.coverPath || ''
  } else {
    categoryId.value = ''
    categoryName.value = ''
    coverPath.value = ''
  }
}, { immediate: true })

async function selectCover() {
  const path = await window.go.app.App.SelectImage()
  if (path) {
    coverPath.value = path
  }
}

async function handleSave() {
  if (!categoryName.value.trim()) return

  try {
    const existingCategory = tabsStore.categories.find(c => c.id === categoryId.value)

    await tabsStore.addCategory({
      id: categoryId.value,
      name: categoryName.value.trim(),
      parentId: categoryId.value
        ? existingCategory?.parentId || ''
        : tabsStore.currentCategoryId,
      coverPath: coverPath.value
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
      <h2>{{ categoryId ? t('category.editCategory') : t('category.newCategory') }}</h2>

      <form @submit.prevent="handleSave">
        <div class="form-group">
          <label for="cat-name">{{ t('category.name') }}</label>
          <input
            id="cat-name"
            type="text"
            v-model="categoryName"
            required
            autofocus
          />
        </div>

        <div class="form-group">
          <label>{{ t('category.coverImage') }}</label>
          <div class="cover-input">
            <input type="text" v-model="coverPath" :placeholder="t('category.defaultCover')" readonly />
            <button type="button" class="btn" @click="selectCover">{{ t('category.select') }}</button>
            <button type="button" class="btn" @click="coverPath = ''" v-if="coverPath">{{ t('category.clear') }}</button>
          </div>
        </div>

        <div class="modal-actions">
          <button type="button" class="btn" @click="uiStore.hideCategoryModal">
            {{ t('confirm.cancel') }}
          </button>
          <button type="submit" class="btn primary">
            {{ t('confirm.save') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.cover-input {
  display: flex;
  gap: 0.5rem;
}
.cover-input input {
  flex: 1;
}
</style>
