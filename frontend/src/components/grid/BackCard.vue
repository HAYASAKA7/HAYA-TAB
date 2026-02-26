<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useTabsStore } from '@/stores'

const { t } = useI18n()
const tabsStore = useTabsStore()

function handleClick() {
  tabsStore.goBack()
}
</script>

<template>
  <div
    class="tab-card folder back-folder"
    @click="handleClick"
  >
    <div class="cover-wrapper">
      <span class="icon-back icon-xl"></span>
    </div>
    <div class="info">
      <div class="title">{{ t('nav.back') }}</div>
    </div>
  </div>
</template>
