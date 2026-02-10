<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useTabsStore, useUIStore } from '@/stores'
import { useToast } from '@/composables/useToast'
import type { Tab } from '@/types'

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
  coverPath: '',
  categoryId: ''
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
      coverPath: data.coverPath || '',
      categoryId: data.categoryId || tabsStore.currentCategoryId
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
    coverPath: existing?.coverPath || '',
    categoryId: existing?.categoryId || tabsStore.currentCategoryId,
    country: formData.value.country || 'US',
    language: formData.value.language || 'en_us',
    tag: formData.value.tag || ''
  }

  try {
    if (isEditMode.value) {
      await tabsStore.updateTab(tab)
    } else {
      await tabsStore.addTab(tab, shouldCopy.value)
    }
    showToast('Saved.')
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
      <h2>{{ isEditMode ? 'Edit Tab Metadata' : 'Add New Tab' }}</h2>

      <form id="edit-form" @submit.prevent="handleSave">
        <input type="hidden" v-model="formData.filePath" />
        <input type="hidden" v-model="shouldCopy" />

        <div class="form-group">
          <label for="edit-title">Title</label>
          <input
            id="edit-title"
            type="text"
            v-model="formData.title"
            required
          />
        </div>

        <div class="form-group">
          <label for="edit-artist">Artist</label>
          <input
            id="edit-artist"
            type="text"
            v-model="formData.artist"
          />
        </div>

        <div class="form-group">
          <label for="edit-album">Album</label>
          <input
            id="edit-album"
            type="text"
            v-model="formData.album"
          />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label for="edit-type">Type</label>
            <select id="edit-type" v-model="formData.type">
              <option value="pdf">PDF</option>
              <option value="gp">Guitar Pro</option>
            </select>
          </div>

          <div class="form-group">
            <label for="edit-tag">Tag</label>
            <input
              id="edit-tag"
              type="text"
              v-model="formData.tag"
              placeholder="e.g. Lead Guitar"
            />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label for="edit-country">Country</label>
            <select id="edit-country" v-model="formData.country">
              <option value="US">US</option>
              <option value="JP">Japan</option>
              <option value="GB">UK</option>
              <option value="DE">Germany</option>
              <option value="FR">France</option>
              <option value="KR">Korea</option>
              <option value="CN">China</option>
            </select>
          </div>

          <div class="form-group">
            <label for="edit-lang">Language</label>
            <select id="edit-lang" v-model="formData.language">
              <option value="en_us">English (US)</option>
              <option value="ja_jp">Japanese</option>
              <option value="en_gb">English (UK)</option>
              <option value="de_de">German</option>
              <option value="fr_fr">French</option>
              <option value="ko_kr">Korean</option>
              <option value="zh_cn">Chinese</option>
            </select>
          </div>
        </div>

        <div class="modal-actions">
          <button type="button" class="btn" @click="uiStore.hideEditModal">
            Cancel
          </button>
          <button type="submit" class="btn primary">
            Save
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
