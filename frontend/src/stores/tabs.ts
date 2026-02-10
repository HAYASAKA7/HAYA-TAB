import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Tab, Category, TabsResponse } from '@/types'

export const useTabsStore = defineStore('tabs', () => {
  // State
  const tabs = ref<Tab[]>([])
  const categories = ref<Category[]>([])
  const currentCategoryId = ref('')
  const loading = ref(false)
  const pagination = ref({
    page: 1,
    pageSize: 50,
    total: 0,
    hasMore: true
  })

  // Search state
  const searchQuery = ref('')
  const searchFilters = ref<string[]>(['title'])
  const searchScope = ref<'global' | 'local'>('local')

  // Batch selection state
  const isBatchSelectMode = ref(false)
  const selectedTabIds = ref<Set<string>>(new Set())

  // Getters
  const currentTabs = computed(() => {
    // Backend handles filtering now
    return tabs.value
  })

  const currentCategories = computed(() => {
    // Hide categories when searching
    if (searchQuery.value) {
      return []
    }
    return categories.value.filter(c => c.parentId === currentCategoryId.value)
  })

  const currentCategory = computed(() => {
    return categories.value.find(c => c.id === currentCategoryId.value)
  })

  const selectedTabs = computed(() => {
    return tabs.value.filter(t => selectedTabIds.value.has(t.id))
  })

  // Actions
  async function fetchTabs() {
    loading.value = true
    try {
      tabs.value = await window.go.main.App.GetTabs() || []
    } catch (err) {
      console.error('Error fetching tabs:', err)
      tabs.value = []
    } finally {
      loading.value = false
    }
  }

  async function fetchTabsPaginated(categoryId?: string) {
    loading.value = true
    try {
      const response: TabsResponse = await window.go.main.App.GetTabsPaginated(
        categoryId ?? currentCategoryId.value,
        pagination.value.page,
        pagination.value.pageSize,
        searchQuery.value,
        searchFilters.value,
        searchScope.value === 'global'
      )
      tabs.value = response.tabs
      pagination.value.total = response.total
      pagination.value.hasMore = response.hasMore
    } catch (err) {
      console.error('Error fetching paginated tabs:', err)
      tabs.value = []
    } finally {
      loading.value = false
    }
  }

  async function loadMore() {
    if (!pagination.value.hasMore || loading.value) return

    pagination.value.page++
    loading.value = true
    try {
      const response: TabsResponse = await window.go.main.App.GetTabsPaginated(
        currentCategoryId.value,
        pagination.value.page,
        pagination.value.pageSize,
        searchQuery.value,
        searchFilters.value,
        searchScope.value === 'global'
      )
      tabs.value = [...tabs.value, ...response.tabs]
      pagination.value.hasMore = response.hasMore
    } catch (err) {
      console.error('Error loading more tabs:', err)
      pagination.value.page--
    } finally {
      loading.value = false
    }
  }

  function setSearchQuery(query: string) {
    searchQuery.value = query
    pagination.value.page = 1
    fetchTabsPaginated()
  }

  function setSearchFilters(filters: string[]) {
    searchFilters.value = filters
    if (searchQuery.value) {
      pagination.value.page = 1
      fetchTabsPaginated()
    }
  }

  function setSearchScope(scope: 'global' | 'local') {
    searchScope.value = scope
    if (searchQuery.value) {
      pagination.value.page = 1
      fetchTabsPaginated()
    }
  }

  async function fetchCategories() {
    try {
      categories.value = await window.go.main.App.GetCategories() || []
    } catch (err) {
      console.error('Error fetching categories:', err)
      categories.value = []
    }
  }

  async function refreshData() {
    await Promise.all([fetchTabsPaginated(), fetchCategories()])
  }

  async function addTab(tab: Tab, shouldCopy: boolean) {
    await window.go.main.App.SaveTab(tab, shouldCopy)
    await refreshData()
  }

  async function updateTab(tab: Tab) {
    await window.go.main.App.UpdateTab(tab)
    await refreshData()
  }

  async function deleteTab(id: string) {
    await window.go.main.App.DeleteTab(id)
    await refreshData()
  }

  async function moveTab(id: string, categoryId: string) {
    await window.go.main.App.MoveTab(id, categoryId)
    await refreshData()
  }

  async function batchDeleteTabs() {
    if (selectedTabIds.value.size === 0) return 0
    const ids = Array.from(selectedTabIds.value)
    const deleted = await window.go.main.App.BatchDeleteTabs(ids)
    exitBatchSelectMode()
    await refreshData()
    return deleted
  }

  async function batchMoveTabs(categoryId: string) {
    if (selectedTabIds.value.size === 0) return 0
    const ids = Array.from(selectedTabIds.value)
    const moved = await window.go.main.App.BatchMoveTabs(ids, categoryId)
    exitBatchSelectMode()
    await refreshData()
    return moved
  }

  async function addCategory(category: Category) {
    await window.go.main.App.AddCategory(category)
    await fetchCategories()
  }

  async function deleteCategory(id: string) {
    await window.go.main.App.DeleteCategory(id)
    await refreshData()
  }

  async function moveCategory(id: string, newParentId: string) {
    await window.go.main.App.MoveCategory(id, newParentId)
    await fetchCategories()
  }

  function navigateToCategory(categoryId: string) {
    currentCategoryId.value = categoryId
    pagination.value.page = 1
    pagination.value.hasMore = true
  }

  function goHome() {
    navigateToCategory('')
  }

  function goBack() {
    const current = currentCategory.value
    navigateToCategory(current?.parentId || '')
  }

  // Batch selection methods
  function toggleBatchSelectMode() {
    isBatchSelectMode.value = !isBatchSelectMode.value
    if (!isBatchSelectMode.value) {
      selectedTabIds.value.clear()
    }
  }

  function exitBatchSelectMode() {
    isBatchSelectMode.value = false
    selectedTabIds.value.clear()
  }

  function toggleTabSelection(tabId: string) {
    if (selectedTabIds.value.has(tabId)) {
      selectedTabIds.value.delete(tabId)
    } else {
      selectedTabIds.value.add(tabId)
    }
    // Trigger reactivity
    selectedTabIds.value = new Set(selectedTabIds.value)
  }

  function selectAllTabs() {
    if (selectedTabIds.value.size === currentTabs.value.length) {
      selectedTabIds.value.clear()
    } else {
      currentTabs.value.forEach(t => selectedTabIds.value.add(t.id))
    }
    selectedTabIds.value = new Set(selectedTabIds.value)
  }

  function isTabSelected(tabId: string) {
    return selectedTabIds.value.has(tabId)
  }

  function getTabById(id: string) {
    return tabs.value.find(t => t.id === id)
  }

  function getCategoryPath(categoryId: string): string[] {
    const path: string[] = []
    let current = categories.value.find(c => c.id === categoryId)
    while (current) {
      path.unshift(current.name)
      current = categories.value.find(c => c.id === current!.parentId)
    }
    return path
  }

  return {
    // State
    tabs,
    categories,
    currentCategoryId,
    loading,
    pagination,
    isBatchSelectMode,
    selectedTabIds,
    searchQuery,
    searchFilters,
    searchScope,

    // Getters
    currentTabs,
    currentCategories,
    currentCategory,
    selectedTabs,

    // Actions
    fetchTabs,
    fetchTabsPaginated,
    loadMore,
    setSearchQuery,
    setSearchFilters,
    setSearchScope,
    fetchCategories,
    refreshData,
    addTab,
    updateTab,
    deleteTab,
    moveTab,
    batchDeleteTabs,
    batchMoveTabs,
    addCategory,
    deleteCategory,
    moveCategory,
    navigateToCategory,
    goHome,
    goBack,
    toggleBatchSelectMode,
    exitBatchSelectMode,
    toggleTabSelection,
    selectAllTabs,
    isTabSelected,
    getTabById,
    getCategoryPath
  }
})
