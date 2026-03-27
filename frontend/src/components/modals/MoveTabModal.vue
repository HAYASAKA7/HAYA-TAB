<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { SYSTEM_CLOUD_CATEGORY_ID } from '@/types'

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showToast, showErrorToast } = useToast()

const selectedCategoryId = ref('')

const sortedCategories = computed(() => {
  return [...tabsStore.categories]
    .filter(c => c.id !== SYSTEM_CLOUD_CATEGORY_ID)
    .sort((a, b) => {
    const pathA = tabsStore.getCategoryPath(a.id).join('/')
    const pathB = tabsStore.getCategoryPath(b.id).join('/')
    return pathA.localeCompare(pathB)
  })
})

async function handleSave() {
  try {
    await tabsStore.addTabToCategory(uiStore.moveModalTabId, selectedCategoryId.value)
    showToast(t('batch.addedToCategory'))
    uiStore.hideMoveModal()
  } catch (err) {
    showErrorToast(err)
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
      <h2>{{ t('batch.addToCategory') }}</h2>

      <form @submit.prevent="handleSave">
        <div class="form-group">
          <label for="move-select">{{ t('batch.selectCategory') }}</label>
          <select id="move-select" v-model="selectedCategoryId">
            <option value="">{{ t('batch.root') }}</option>
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
            {{ t('confirm.cancel') }}
          </button>
          <button type="submit" class="btn primary">
            {{ t('batch.add') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
