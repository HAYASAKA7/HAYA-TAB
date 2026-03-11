import { callBackend } from './api';

/**
 * Service for Cloud (WebDAV) operations.
 */
export const CloudService = {
  /**
   * Check WebDAV connection status.
   */
  async checkStatus(): Promise<boolean> {
    return await callBackend<boolean>('WebDAVCheckStatus');
  },

  /**
   * Test a WebDAV connection with provided credentials.
   */
  async testConnection(url: string, user: string, pass: string): Promise<void> {
    return await callBackend<void>('WebDAVTestConnection', url, user, pass);
  },

  /**
   * Initialize WebDAV client with current settings.
   */
  async initialize(): Promise<void> {
    return await callBackend<void>('WebDAVInitialize');
  },

  /**
   * Reconnect WebDAV client.
   */
  async reconnect(): Promise<void> {
    return await callBackend<void>('WebDAVReconnect');
  },

  /**
   * List remote directories for folder selection.
   */
  async listRemoteDirectories(url: string, user: string, pass: string, path: string): Promise<string[]> {
    return await callBackend<string[]>('WebDAVListRemoteDirectories', url, user, pass, path) || [];
  },

  /**
   * List files and directories at a remote path.
   */
  async listDir(url: string, user: string, pass: string, path: string): Promise<any[]> {
    return await callBackend<any[]>('WebDAVListDir', url, user, pass, path) || [];
  },

  /**
   * Scan remote directory recursively for music tab files.
   */
  async scanRemoteFiles(url: string, user: string, pass: string, path: string): Promise<any[]> {
    return await callBackend<any[]>('WebDAVScanRemoteFiles', url, user, pass, path) || [];
  },

  /**
   * Upload local files to a remote path.
   */
  async uploadFiles(url: string, user: string, pass: string, localPaths: string[], remotePath: string): Promise<void> {
    return await callBackend<void>('WebDAVUploadFiles', url, user, pass, localPaths, remotePath);
  },

  /**
   * Download remote files to local storage and link them.
   */
  async downloadFiles(url: string, user: string, pass: string, remotePaths: string[]): Promise<void> {
    return await callBackend<void>('WebDAVDownloadFiles', url, user, pass, remotePaths);
  },

  /**
   * Add online files to the library without downloading (metadata only).
   */
  async addOnlineFiles(url: string, user: string, pass: string, remotePaths: string[]): Promise<void> {
    return await callBackend<void>('WebDAVAddOnlineFiles', url, user, pass, remotePaths);
  },

  /**
   * Discover virtual volumes from the database.
   */
  async discoverVolumes(): Promise<any[]> {
    return await callBackend<any[]>('WebDAVDiscoverVolumes') || [];
  },

  /**
   * Create a new virtual volume.
   */
  async createVolume(name: string, remotePath: string): Promise<any> {
    return await callBackend<any>('WebDAVCreateVolume', name, remotePath);
  },

  /**
   * Check health of all virtual volumes.
   */
  async checkVolumeHealth(): Promise<Record<string, boolean>> {
    return await callBackend<Record<string, boolean>>('WebDAVCheckVolumeHealth') || {};
  },

  /**
   * Get count of orphaned cloud tabs.
   */
  async getOrphanedTabsCount(): Promise<number> {
    return await callBackend<number>('WebDAVGetOrphanedTabsCount');
  },

  /**
   * Cleanup orphaned cloud tabs.
   */
  async cleanupOrphanedTabs(): Promise<number> {
    return await callBackend<number>('WebDAVCleanupOrphanedTabs');
  },

  /**
   * Migrate cloud tabs to new volume structure.
   */
  async migrateCloudTabs(): Promise<void> {
    return await callBackend<void>('WebDAVMigrateCloudTabs');
  }
};
