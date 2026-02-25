<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showToast } = useToast()

const selectedCount = computed(() => tabsStore.selectedTabIds.size)
const isVisible = computed(() => tabsStore.isBatchSelectMode && selectedCount.value > 0)

async function handleDelete() {
  const selectedTabs = tabsStore.selectedTabs
  const managedCount = selectedTabs.filter((tab: any) => tab.isManaged).length
  const linkedCount = selectedTabs.length - managedCount

  let message = `${t('batch.removeConfirm')} <strong>${selectedCount.value}</strong> ${t('batch.tabs')}.`

  if (managedCount > 0 && linkedCount > 0) {
    message += `<ul>
      <li><strong>${managedCount}</strong> ${t('batch.uploadedWillDelete')}</li>
      <li><strong>${linkedCount}</strong> ${t('batch.linkedWillUnlink')}</li>
    </ul>`
  } else if (managedCount > 0) {
    message += `<br><br><strong>${managedCount}</strong> ${t('batch.uploadedWillDelete')}.`
  } else {
    message += `<br><br><strong>${linkedCount}</strong> ${t('batch.linkedWillUnlink')}.`
  }

  uiStore.showConfirmModal(t('batch.removeTabs'), message, t('confirm.remove'), true, async () => {
    const deleted = await tabsStore.batchDeleteTabs()
    showToast(t('batch.removed', { count: deleted }))
  })
}

function handleMove() {
  uiStore.showBatchMoveModal()
}
</script>

<template>
  <div
    id="batch-action-bar"
    :class="{ hidden: !isVisible }"
  >
    <div class="batch-info">
      <span id="batch-selected-count">{{ selectedCount }}</span> {{ t('batch.selected') }}
      <button class="btn small" @click="tabsStore.selectAllTabs">{{ t('batch.selectAll') }}</button>
    </div>
    <div class="batch-actions">
      <button class="btn" @click="handleMove">
        <span class="icon-folder"></span> {{ t('batch.moveTo') }}
      </button>
      <button class="btn" @click="uiStore.showCloudUploadModal(Array.from(tabsStore.selectedTabIds).map(id => tabsStore.tabs.find(t => t.id === id)?.filePath).filter(p => !!p) as string[])">
        <span class="icon-cloud"></span> {{ t('cloud.upload') }}
      </button>
      <button class="btn danger" @click="handleDelete">
        <span class="icon-trash"></span> {{ t('batch.remove') }}
      </button>
    </div>
  </div>
</template>
