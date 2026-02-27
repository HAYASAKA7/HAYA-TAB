<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import { SYSTEM_CLOUD_CATEGORY_ID } from '@/types'

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showToast } = useToast()

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
    const added = await tabsStore.batchAddTabsToCategory(selectedCategoryId.value)
    showToast(t('batch.addedTabs', { count: added }))
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
      <h2>{{ t('batch.addSelectedToCategory') }}</h2>

      <form @submit.prevent="handleSave">
        <div class="form-group">
          <label for="batch-move-select">{{ t('batch.selectCategory') }}</label>
          <select id="batch-move-select" v-model="selectedCategoryId">
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
          <button type="button" class="btn" @click="uiStore.hideBatchMoveModal">
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
