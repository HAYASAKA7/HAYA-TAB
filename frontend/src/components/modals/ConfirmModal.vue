<script setup lang="ts">
import { useUIStore } from '@/stores/ui'

const uiStore = useUIStore()

function handleConfirm() {
  if (uiStore.confirmModalData?.onConfirm) {
    uiStore.confirmModalData.onConfirm()
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
      <p id="confirm-message" v-html="uiStore.confirmModalData?.message"></p>
      <div class="modal-actions">
        <button
          id="confirm-cancel-btn"
          class="btn"
          @click="uiStore.hideConfirmModal"
        >
          Cancel
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
