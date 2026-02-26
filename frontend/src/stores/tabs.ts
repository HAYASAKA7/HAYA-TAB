import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Tab, Category, TabsResponse } from '@/types'
import { SYSTEM_CLOUD_CATEGORY_ID } from '@/types'

export const useTabsStore = defineStore('tabs', () => {
  // State
  const tabs = ref<Tab[]>([])
  const categories = ref<Category[]>([])
  const recentCategories = ref<Category[]>([])
  const recentTabs = ref<Tab[]>([])
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
  const sortBy = ref('title')
  const sortDesc = ref(false)

  // Batch selection state
  const isBatchSelectMode = ref(false)
  const selectedTabIds = ref<Set<string>>(new Set())

  // Getters
  const currentTabs = computed(() => {
    // Backend handles filtering now
    return tabs.value
  })

  const currentCategories = computed(() => {
    // If there is a search query, perform fuzzy search on category names
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase()
      return categories.value.filter(c => c.name.toLowerCase().includes(q))
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
        searchScope.value === 'global',
        sortBy.value,
        sortDesc.value
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
        searchScope.value === 'global',
        sortBy.value,
        sortDesc.value
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

  function setSort(by: string, desc: boolean) {
    sortBy.value = by
    sortDesc.value = desc
    pagination.value.page = 1
    fetchTabsPaginated()
  }

  async function fetchCategories() {
    try {
      const result = await window.go.main.App.GetCategories() || []
      // Sort categories: cloud category first, then alphabetically
      categories.value = result.sort((a: Category, b: Category) => {
        if (a.id === SYSTEM_CLOUD_CATEGORY_ID) return -1
        if (b.id === SYSTEM_CLOUD_CATEGORY_ID) return 1
        return a.name.localeCompare(b.name)
      })
    } catch (err) {
      console.error('Error fetching categories:', err)
      categories.value = []
    }
  }

  async function fetchRecentCategories(limit: number) {
    loading.value = true
    try {
      recentCategories.value = await window.go.main.App.GetRecentCategories(limit) || []
    } catch (err) {
      console.error('Error fetching recent categories:', err)
      recentCategories.value = []
    } finally {
      loading.value = false
    }
  }

  async function fetchRecentTabs(limit: number) {
    loading.value = true
    try {
      // @ts-ignore
      recentTabs.value = await window.go.main.App.GetRecentTabs(limit) || []
    } catch (err) {
      console.error('Error fetching recent tabs:', err)
      recentTabs.value = []
    } finally {
      loading.value = false
    }
  }

  async function refreshData() {
    await Promise.all([fetchTabsPaginated(), fetchCategories()])
  }

  async function addTab(tab: Tab, shouldCopy: boolean) {
    const savedTab = await window.go.main.App.SaveTab(tab, shouldCopy)
    // Add in-place to preserve scroll position
    if (savedTab) {
      addTabsInPlace([savedTab])
    }
  }

  // Add tabs in-place without full refresh (preserves scroll position)
  function addTabsInPlace(newTabs: Tab[]) {
    // Prepend new tabs to the beginning of the list
    tabs.value = [...newTabs, ...tabs.value]
    pagination.value.total += newTabs.length
    // Also add to recentTabs if it's populated
    if (recentTabs.value.length > 0) {
      recentTabs.value = [...newTabs, ...recentTabs.value]
    }
  }

  async function updateTab(tab: Tab) {
    await window.go.main.App.UpdateTab(tab)
    await refreshData()
  }

  // In-place update of a single tab without full refresh (preserves scroll position)
  function updateTabInPlace(tabId: string, updates: Partial<Tab>) {
    const targetTab = tabs.value.find(t => t.id === tabId)
    if (targetTab) {
      Object.assign(targetTab, updates)
    }
    // Also update in recentTabs if present
    const recentTab = recentTabs.value.find(t => t.id === tabId)
    if (recentTab) {
      Object.assign(recentTab, updates)
    }
  }

  async function deleteTab(id: string) {
    await window.go.main.App.DeleteTab(id)
    await refreshData()
  }

  // Delete tab and remove from local state without full refresh (preserves scroll position)
  async function deleteTabInPlace(id: string) {
    await window.go.main.App.DeleteTab(id)
    // Remove from tabs array in-place
    const tabIndex = tabs.value.findIndex(t => t.id === id)
    if (tabIndex !== -1) {
      tabs.value.splice(tabIndex, 1)
      pagination.value.total = Math.max(0, pagination.value.total - 1)
    }
    // Also remove from recentTabs if present
    const recentIndex = recentTabs.value.findIndex(t => t.id === id)
    if (recentIndex !== -1) {
      recentTabs.value.splice(recentIndex, 1)
    }
  }

  async function moveTab(id: string, categoryId: string) {
    await window.go.main.App.MoveTab(id, categoryId)
    await refreshData()
  }

  async function addTabToCategory(id: string, categoryId: string) {
    await window.go.main.App.AddTabToCategory(id, categoryId)
    await refreshData()
  }

  async function removeTabFromCategory(id: string, categoryId: string) {
    await window.go.main.App.RemoveTabFromCategory(id, categoryId)
    // Remove tab from view in-place if we're in that category (preserves scroll position)
    if (currentCategoryId.value === categoryId) {
      const tabIndex = tabs.value.findIndex(t => t.id === id)
      if (tabIndex !== -1) {
        tabs.value.splice(tabIndex, 1)
        pagination.value.total = Math.max(0, pagination.value.total - 1)
      }
    }
  }

  async function batchDeleteTabs() {
    if (selectedTabIds.value.size === 0) return 0
    const ids = Array.from(selectedTabIds.value)
    const deleted = await window.go.main.App.BatchDeleteTabs(ids)
    exitBatchSelectMode()
    // Remove deleted tabs in-place to preserve scroll position
    tabs.value = tabs.value.filter(t => !ids.includes(t.id))
    recentTabs.value = recentTabs.value.filter(t => !ids.includes(t.id))
    pagination.value.total = Math.max(0, pagination.value.total - deleted)
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

  async function batchAddTabsToCategory(categoryId: string) {
    if (selectedTabIds.value.size === 0) return 0
    const ids = Array.from(selectedTabIds.value)
    // @ts-ignore
    const added = await window.go.main.App.BatchAddTabsToCategory(ids, categoryId)
    exitBatchSelectMode()
    await refreshData()
    return added
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
    fetchTabsPaginated()
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
    recentCategories,
    recentTabs,
    currentCategoryId,
    loading,
    pagination,
    isBatchSelectMode,
    selectedTabIds,
    searchQuery,
    searchFilters,
    searchScope,
    sortBy,
    sortDesc,

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
    setSort,
    fetchCategories,
    fetchRecentCategories,
    fetchRecentTabs,
    refreshData,
    addTab,
    addTabsInPlace,
    updateTab,
    deleteTab,
    deleteTabInPlace,
    moveTab,
    addTabToCategory,
    removeTabFromCategory,
    batchDeleteTabs,
    batchMoveTabs,
    batchAddTabsToCategory,
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
    getCategoryPath,
    updateTabInPlace
  }
})
