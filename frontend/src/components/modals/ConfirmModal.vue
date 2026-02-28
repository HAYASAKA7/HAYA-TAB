<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useUIStore } from '@/stores/ui'

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
  <div
    v-if="uiStore.confirmModalVisible"
    id="confirm-modal"
    class="modal-overlay"
    @click.self="uiStore.hideConfirmModal"
  >
    <div class="modal confirm-modal">
      <h2 id="confirm-title">{{ uiStore.confirmModalData?.title }}</h2>
      <p id="confirm-message" class="selectable" v-html="uiStore.confirmModalData?.message"></p>
      <div class="modal-actions" style="justify-content: flex-end;">
        <div style="margin-right: auto;" v-if="uiStore.confirmModalData?.altText">
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
      </div>
    </div>
  </div>
</template>
