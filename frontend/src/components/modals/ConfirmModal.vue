<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useUIStore } from '@/stores/ui'
import BaseModal from '@/components/common/BaseModal.vue'

const { t } = useI18n()
const uiStore = useUIStore()

function handleConfirm() {
  if (uiStore.confirmModalData?.onConfirm) {
    uiStore.confirmModalData.onConfirm()
  }
  uiStore.hideConfirmModal()
}

function handleAlt() {
  if (uiStore.confirmModalData?.onAlt) {
    uiStore.confirmModalData.onAlt()
  }
  uiStore.hideConfirmModal()
}
</script>

<template>
  <BaseModal
    :open="uiStore.confirmModalVisible"
    :title="uiStore.confirmModalData?.title || ''"
    size="small"
    @close="uiStore.hideConfirmModal"
  >
    <p id="confirm-message" class="selectable" v-html="uiStore.confirmModalData?.message"></p>

    <template #actions>
      <div v-if="uiStore.confirmModalData?.altText" class="modal-actions__leading">
        <button class="btn primary" @click="handleAlt">
          {{ uiStore.confirmModalData?.altText }}
        </button>
      </div>
      <button
        id="confirm-cancel-btn"
        class="btn"
        @click="uiStore.hideConfirmModal"
      >
        {{ t('confirm.cancel') }}
      </button>
      <button
        id="confirm-ok-btn"
        :class="['btn', uiStore.confirmModalData?.isDanger ? 'danger' : 'primary']"
        @click="handleConfirm"
      >
        {{ uiStore.confirmModalData?.okText }}
      </button>
    </template>
  </BaseModal>
</template>
