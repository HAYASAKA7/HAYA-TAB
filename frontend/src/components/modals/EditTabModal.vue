<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import type { Tab } from '@/types'

const { t } = useI18n()
const tabsStore = useTabsStore()
const uiStore = useUIStore()
const { showToast } = useToast()

const isEditMode = computed(() => !!uiStore.editModalData?.id && tabsStore.tabs.some(t => t.id === uiStore.editModalData?.id))

// Form data
const formData = ref<Partial<Tab>>({
  id: '',
  title: '',
  artist: '',
  album: '',
  filePath: '',
  type: 'pdf',
  country: 'US',
  language: 'en_us',
  tag: '',
  isManaged: false,
  isCloud: false,
  coverPath: '',
  categoryIds: [] as string[]
})

const shouldCopy = ref(false)

// Watch for modal data changes
watch(() => uiStore.editModalData, (data) => {
  if (data) {
    formData.value = {
      id: data.id || '',
      title: data.title || '',
      artist: data.artist || '',
      album: data.album || '',
      filePath: data.filePath || '',
      type: data.type || 'pdf',
      country: data.country || 'US',
      language: data.language || 'en_us',
      tag: data.tag || '',
      isManaged: data.isManaged || false,
      isCloud: data.isCloud || false,
      coverPath: data.coverPath || '',
      categoryIds: data.categoryIds || (data.categoryId ? [data.categoryId] : []) || (tabsStore.currentCategoryId ? [tabsStore.currentCategoryId] : [])
    }
    shouldCopy.value = false
  }
}, { immediate: true })

async function handleSave() {
  const existing = tabsStore.tabs.find(t => t.id === formData.value.id)

  const tab: Tab = {
    id: formData.value.id || '',
    title: formData.value.title || '',
    artist: formData.value.artist || '',
    album: formData.value.album || '',
    filePath: formData.value.filePath || '',
    type: (formData.value.type as 'pdf' | 'gp' | 'unknown') || 'pdf',
    isManaged: existing?.isManaged || false,
    isCloud: existing?.isCloud || false,
    coverPath: existing?.coverPath || '',
    categoryIds: existing?.categoryIds || (tabsStore.currentCategoryId ? [tabsStore.currentCategoryId] : []),
    country: formData.value.country || 'US',
    language: formData.value.language || 'en_us',
    originCountry: existing?.originCountry || '',
    tag: formData.value.tag || '',
    addedAt: existing?.addedAt || 0,
    lastOpened: existing?.lastOpened || 0,
    initialAz: existing?.initialAz || '#',   // Will be recalculated by backend
    initialKana: existing?.initialKana || '#' // Will be recalculated by backend
  }

  try {
    if (isEditMode.value) {
      await tabsStore.updateTab(tab)
    } else {
      await tabsStore.addTab(tab, shouldCopy.value)
    }
    showToast(t('toast.saved'))
    uiStore.hideEditModal()
  } catch (err) {
    showToast(String(err), 'error')
  }
}
</script>

<template>
  <div
    v-if="uiStore.editModalVisible"
    id="modal-overlay"
    class="modal-overlay"
    @click.self="uiStore.hideEditModal"
  >
    <div class="modal">
      <h2>{{ isEditMode ? t('tab.editMetadata') : t('tab.addNewTab') }}</h2>

      <form id="edit-form" @submit.prevent="handleSave">
        <input type="hidden" v-model="formData.filePath" />
        <input type="hidden" v-model="shouldCopy" />

        <div class="form-group">
          <label for="edit-title">{{ t('tab.title') }}</label>
          <input
            id="edit-title"
            type="text"
            v-model="formData.title"
            required
          />
        </div>

        <div class="form-group">
          <label for="edit-artist">{{ t('tab.artist') }}</label>
          <input
            id="edit-artist"
            type="text"
            v-model="formData.artist"
          />
        </div>

        <div class="form-group">
          <label for="edit-album">{{ t('tab.album') }}</label>
          <input
            id="edit-album"
            type="text"
            v-model="formData.album"
          />
        </div>

        <div class="form-group">
          <label for="edit-type">{{ t('tab.type') }}</label>
          <select id="edit-type" v-model="formData.type">
            <option value="pdf">{{ t('tab.pdf') }}</option>
            <option value="gp">{{ t('tab.guitarPro') }}</option>
          </select>
        </div>

        <div class="form-group">
          <label for="edit-tag">{{ t('tab.tag') }}</label>
          <input
            id="edit-tag"
            type="text"
            v-model="formData.tag"
            :placeholder="t('tab.tagPlaceholder')"
          />
        </div>

        <div class="form-group">
          <label for="edit-country">{{ t('tab.country') }}</label>
          <select id="edit-country" v-model="formData.country">
            <option value="US">{{ t('countries.US') }}</option>
            <option value="JP">{{ t('countries.JP') }}</option>
            <option value="GB">{{ t('countries.GB') }}</option>
            <option value="DE">{{ t('countries.DE') }}</option>
            <option value="FR">{{ t('countries.FR') }}</option>
            <option value="KR">{{ t('countries.KR') }}</option>
            <option value="CN">{{ t('countries.CN') }}</option>
          </select>
        </div>

        <div class="form-group">
          <label for="edit-lang">{{ t('tab.language') }}</label>
          <select id="edit-lang" v-model="formData.language">
            <option value="en_us">{{ t('languages.en_us') }}</option>
            <option value="ja_jp">{{ t('languages.ja_jp') }}</option>
            <option value="en_gb">{{ t('languages.en_gb') }}</option>
            <option value="de_de">{{ t('languages.de_de') }}</option>
            <option value="fr_fr">{{ t('languages.fr_fr') }}</option>
            <option value="ko_kr">{{ t('languages.ko_kr') }}</option>
            <option value="zh_cn">{{ t('languages.zh_cn') }}</option>
          </select>
        </div>

        <div class="modal-actions">
          <button type="button" class="btn" @click="uiStore.hideEditModal">
            {{ t('confirm.cancel') }}
          </button>
          <button type="submit" class="btn primary">
            {{ t('confirm.save') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
