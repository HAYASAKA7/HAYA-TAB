import { callBackend } from './api';
import type { Tab, TabsResponse } from '@/types';

/**
 * Service for Tab operations.
 */
export const TabService = {
  /**
   * Fetch all tabs.
   */
  async getTabs(): Promise<Tab[]> {
    return await callBackend<Tab[]>('GetTabs') || [];
  },

  /**
   * Fetch paginated tabs with filters and sorting.
   */
  async getTabsPaginated(
    categoryId: string,
    page: number,
    pageSize: number,
    searchQuery: string,
    filterBy: string[],
    isGlobal: boolean,
    sortBy: string,
    sortDesc: boolean
  ): Promise<TabsResponse> {
    return await callBackend<TabsResponse>(
      'GetTabsPaginated',
      categoryId,
      page,
      pageSize,
      searchQuery,
      filterBy,
      isGlobal,
      sortBy,
      sortDesc
    );
  },

  /**
   * Fetch a tab by its ID.
   */
  async getTabById(id: string): Promise<Tab> {
    return await callBackend<Tab>('GetTabById', id);
  },

  /**
   * Fetch the raw content of a tab file.
   */
  async getTabContent(id: string): Promise<string> {
    return await callBackend<string>('GetTabContent', id);
  },

  /**
   * Fetch recent tabs.
   */
  async getRecentTabs(limit: number): Promise<Tab[]> {
    return await callBackend<Tab[]>('GetRecentTabs', limit) || [];
  },

  /**
   * Save a tab.
   */
  async saveTab(tab: Tab, shouldCopy: boolean): Promise<Tab> {
    return await callBackend<Tab>('SaveTab', tab, shouldCopy);
  },

  /**
   * Update a tab's metadata.
   */
  async updateTab(tab: Tab): Promise<void> {
    return await callBackend<void>('UpdateTab', tab);
  },

  /**
   * Update tab metadata (specific to Guitar Pro files).
   */
  async updateTabMetadata(id: string, title: string, artist: string, album: string): Promise<void> {
    return await callBackend<void>('UpdateTabMetadata', id, title, artist, album);
  },

  /**
   * Delete a tab.
   */
  async deleteTab(id: string): Promise<void> {
    return await callBackend<void>('DeleteTab', id);
  },

  /**
   * Batch delete tabs.
   */
  async batchDeleteTabs(ids: string[]): Promise<number> {
    return await callBackend<number>('BatchDeleteTabs', ids);
  },

  /**
   * Batch move tabs.
   */
  async batchMoveTabs(ids: string[], categoryId: string): Promise<number> {
    return await callBackend<number>('BatchMoveTabs', ids, categoryId);
  },

  /**
   * Batch add tabs to a category.
   */
  async batchAddTabsToCategory(ids: string[], categoryId: string): Promise<number> {
    return await callBackend<number>('BatchAddTabsToCategory', ids, categoryId);
  },

  /**
   * Move a tab to a different category.
   */
  async moveTab(id: string, categoryId: string): Promise<void> {
    return await callBackend<void>('MoveTab', id, categoryId);
  },

  /**
   * Update a tab's category list directly.
   */
  async updateTabCategories(id: string, categoryIds: string[]): Promise<void> {
    return await callBackend<void>('UpdateTabCategories', id, categoryIds);
  },

  /**
   * Add a tab to an additional category.
   */
  async addTabToCategory(id: string, categoryId: string): Promise<void> {
    return await callBackend<void>('AddTabToCategory', id, categoryId);
  },

  /**
   * Remove a tab from a category.
   */
  async removeTabFromCategory(id: string, categoryId: string): Promise<void> {
    return await callBackend<void>('RemoveTabFromCategory', id, categoryId);
  },

  /**
   * Open a tab with the system default or internal viewer.
   */
  async openTab(id: string): Promise<void> {
    return await callBackend<void>('OpenTab', id);
  },

  /**
   * Mark a tab as opened (updates lastOpened timestamp).
   */
  async markAsOpened(id: string): Promise<void> {
    return await callBackend<void>('MarkAsOpened', id);
  },

  /**
   * Recalculate initials for all tabs.
   */
  async recalculateAllInitials(): Promise<number> {
    return await callBackend<number>('RecalculateAllInitials');
  },

  /**
   * Export a tab to a specified folder.
   */
  async exportTab(id: string, destFolder: string): Promise<void> {
    return await callBackend<void>('ExportTab', id, destFolder);
  },

  /**
   * Download a cloud tab to local storage.
   */
  async downloadCloudTabToLocal(id: string): Promise<void> {
    return await callBackend<void>('DownloadCloudTabToLocal', id);
  },

  /**
   * Get the base64 encoded cover image for a given path.
   */
  async getCover(path: string): Promise<string> {
    return await callBackend<string>('GetCover', path);
  },

  /**
   * Save annotations for a PDF page.
   */
  async saveTabAnnotations(tabId: string, pageNumber: number, jsonData: string): Promise<void> {
    return await callBackend<void>('SaveTabAnnotations', tabId, pageNumber, jsonData);
  },

  /**
   * Get annotations for a PDF page.
   */
  async getTabAnnotations(tabId: string, pageNumber: number): Promise<string> {
    return await callBackend<string>('GetTabAnnotations', tabId, pageNumber);
  }
};
