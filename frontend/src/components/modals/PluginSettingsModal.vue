<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUIStore } from '@/stores'
import { PluginService } from '@/services/PluginService'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const uiStore = useUIStore()
const { showToast } = useToast()

const config = ref<Record<string, string>>({})

watch(() => uiStore.pluginModalVisible, (visible) => {
  if (visible && uiStore.currentPlugin) {
    config.value = { ...uiStore.currentPlugin.config }
  }
})

async function handleSave() {
  if (!uiStore.currentPlugin) return
  
  try {
    await PluginService.updatePluginConfig(
      uiStore.currentPlugin.id, 
      config.value, 
      uiStore.currentPlugin.enabled
    )
    // Update local state
    uiStore.currentPlugin.config = { ...config.value }
    showToast(t('plugins.save_success'), 'success')
    uiStore.hidePluginModal()
  } catch (e) {
    showToast(t('plugins.save_error'), 'error')
  }
}
</script>

<template>
  <div v-if="uiStore.pluginModalVisible" class="modal-overlay" @click.self="uiStore.hidePluginModal">
    <div class="modal">
      <h2>{{ uiStore.currentPlugin?.name }} {{ t('plugins.settings') }}</h2>
      
      <div class="modal-content">
        <div v-for="(type, key) in (uiStore.currentPlugin?.settingsSchema as Record<string, string>)" :key="key" class="form-group">
          <label>{{ key }}</label>
          <input 
            v-if="type === 'password'" 
            type="password" 
            v-model="config[key as string]" 
            :placeholder="key as string"
          >
          <input 
            v-else 
            type="text" 
            v-model="config[key as string]" 
            :placeholder="key as string"
          >
        </div>
      </div>

      <div class="modal-actions">
        <button class="btn" @click="uiStore.hidePluginModal">{{ t('confirm.cancel') }}</button>
        <button class="btn primary" @click="handleSave">{{ t('confirm.save') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-content {
  margin-top: 10px;
  max-height: 60vh;
  overflow-y: auto;
  padding: 5px 8px; /* Added horizontal padding for focus ring */
  margin-left: -4px; /* Offset margin to keep text alignment if needed */
}
</style>
