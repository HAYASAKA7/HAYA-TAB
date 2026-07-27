<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { FileService } from '@/services'
import BaseModal from '@/components/common/BaseModal.vue'

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showErrorToast } = useToast()

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
  const path = await FileService.selectImage()
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
    showErrorToast(err)
  }
}
</script>

<template>
  <BaseModal
    :open="uiStore.categoryModalVisible"
    :title="categoryId ? t('category.editCategory') : t('category.newCategory')"
    size="small"
    @close="uiStore.hideCategoryModal"
  >
    <form id="category-form" @submit.prevent="handleSave">
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
    </form>

    <template #actions>
      <button type="button" class="btn" @click="uiStore.hideCategoryModal">
        {{ t('confirm.cancel') }}
      </button>
      <button type="submit" form="category-form" class="btn primary">
        {{ t('confirm.save') }}
      </button>
    </template>
  </BaseModal>
</template>

<style scoped>
.cover-input {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 0.5rem;
}
.cover-input input {
  min-width: 0;
}
</style>
