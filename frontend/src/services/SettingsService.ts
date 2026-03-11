import { callBackend } from './api';
import type { Settings } from '@/types';

/**
 * Service for Settings and Sync operations.
 */
export const SettingsService = {
  /**
   * Fetch current application settings.
   */
  async getSettings(): Promise<Settings> {
    return await callBackend<Settings>('GetSettings');
  },

  /**
   * Save application settings.
   */
  async saveSettings(settings: Settings): Promise<void> {
    return await callBackend<void>('SaveSettings', settings);
  },

  /**
   * Trigger a manual file synchronization.
   */
  async triggerSync(): Promise<any> {
    return await callBackend<any>('TriggerSync');
  },

  /**
   * Get the port of the local file server.
   */
  async getFileServerPort(): Promise<number> {
    return await callBackend<number>('GetFileServerPort');
  },

  /**
   * Check for data migration status.
   */
  async checkMigration(target: string): Promise<any> {
    return await callBackend<any>('CheckMigration', target);
  },

  /**
   * Migrate data to a new path.
   */
  async migrateData(target: string, newPath: string, copyOnly: boolean): Promise<void> {
    return await callBackend<void>('MigrateData', target, newPath, copyOnly);
  }
};
