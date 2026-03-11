import { callBackend } from './api';
import type { Category } from '@/types';

/**
 * Service for Category operations.
 */
export const CategoryService = {
  /**
   * Fetch all categories.
   */
  async getCategories(): Promise<Category[]> {
    return await callBackend<Category[]>('GetCategories') || [];
  },

  /**
   * Fetch recent categories.
   */
  async getRecentCategories(limit: number): Promise<Category[]> {
    return await callBackend<Category[]>('GetRecentCategories', limit) || [];
  },

  /**
   * Add or update a category (upsert).
   */
  async addCategory(category: Category): Promise<void> {
    return await callBackend<void>('AddCategory', category);
  },

  /**
   * Delete a category.
   */
  async deleteCategory(id: string): Promise<void> {
    return await callBackend<void>('DeleteCategory', id);
  },

  /**
   * Move a category to a different parent category.
   */
  async moveCategory(id: string, newParentId: string): Promise<void> {
    return await callBackend<void>('MoveCategory', id, newParentId);
  }
};
