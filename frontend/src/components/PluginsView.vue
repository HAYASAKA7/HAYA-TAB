<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUIStore } from '@/stores'
import { PluginService, type PluginInfo } from '@/services/PluginService'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const uiStore = useUIStore()
const { showToast } = useToast()
const plugins = ref<PluginInfo[]>([])
const loading = ref(true)

async function loadPlugins() {
  loading.value = true
  try {
    plugins.value = await PluginService.getPlugins()
  } catch (e) {
    console.error('Failed to load plugins:', e)
  } finally {
    loading.value = false
  }
}

async function togglePlugin(plugin: PluginInfo) {
  try {
    await PluginService.updatePluginConfig(plugin.id, { ...plugin.config }, plugin.enabled)
  } catch (e) {
    showToast(t('plugins.toggle_error'), 'error')
    plugin.enabled = !plugin.enabled
  }
}

function openSettings(plugin: PluginInfo) {
  uiStore.showPluginModal(plugin)
}

onMounted(loadPlugins)
</script>

<template>
  <header class="view-header sticky">
    <h1>{{ t('plugins.title') }}</h1>
  </header>
  
  <div class="plugins-view-container">
    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
    </div>

    <div v-else-if="plugins.length === 0" class="empty-state">
      <span class="icon-plugins large-icon"></span>
      <p>{{ t('plugins.no_plugins') }}</p>
    </div>

    <div v-else class="plugin-list">
      <div v-for="p in plugins" :key="p.id" class="plugin-card">
        <div class="plugin-main-row">
          <div class="plugin-info">
            <span class="plugin-name">{{ p.name }}</span>
            <span class="plugin-version">v{{ p.version }}</span>
          </div>
          
          <div class="plugin-actions">
            <button 
              class="btn-icon"
              :disabled="!p.settingsSchema || Object.keys(p.settingsSchema).length === 0"
              @click="openSettings(p)"
              :title="t('plugins.settings')"
            >
              <span class="icon-settings"></span>
            </button>
            
            <div class="toggle-switch">
              <input 
                type="checkbox" 
                :id="'toggle-' + p.id" 
                v-model="p.enabled" 
                @change="togglePlugin(p)"
              >
              <label :for="'toggle-' + p.id"></label>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.plugins-view-container {
  padding: 1.5rem 2rem;
  max-width: 900px;
  margin: 0 auto;
  overflow-y: auto;
  height: calc(100% - 80px);
}

.view-header {
  padding: 1.5rem 2rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 10;
}

.view-header h1 {
  margin: 0;
  font-size: 1.8rem;
}

.plugin-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.plugin-card {
  background: var(--card-bg);
  border-radius: 8px;
  padding: 16px 20px;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
}

.plugin-main-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.plugin-info {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.plugin-name {
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--text-primary);
}

.plugin-version {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.plugin-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

.plugin-config-panel {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-group {
  margin-bottom: 12px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-weight: 500;
  font-size: 0.9rem;
}

.form-group input {
  width: 100%;
}

.config-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.loading-state, .empty-state {
  text-align: center;
  padding: 4rem;
  color: var(--text-muted);
}

.large-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.btn-icon {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 8px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-icon:hover:not(:disabled) {
  background: var(--hover);
  color: var(--text);
}

.btn-icon.active {
  background: var(--primary);
  color: white;
}

.btn-icon:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

/* Toggle Switch Styles */
.toggle-switch {
  position: relative;
  width: 40px;
  height: 20px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-switch label {
  position: absolute;
  cursor: pointer;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: var(--bg-tertiary, #34495e);
  transition: .3s;
  border-radius: 20px;
}

.toggle-switch label:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 3px;
  bottom: 3px;
  background-color: #fff;
  transition: .3s;
  border-radius: 50%;
}

.toggle-switch input:checked + label {
  background-color: var(--primary, #965233);
}

.toggle-switch input:checked + label:before {
  transform: translateX(20px);
}
</style>
