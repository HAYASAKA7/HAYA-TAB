import { callBackend } from './api';
import type { Tab } from '@/types';

/**
 * Service for file operations (dialogs, processing).
 */
export const FileService = {
  /**
   * Open a file selection dialog.
   */
  async selectFiles(): Promise<string[]> {
    return await callBackend<string[]>('SelectFiles') || [];
  },

  /**
   * Open a folder selection dialog.
   */
  async selectFolder(): Promise<string> {
    return await callBackend<string>('SelectFolder');
  },

  /**
   * Open an image selection dialog.
   */
  async selectImage(): Promise<string> {
    return await callBackend<string>('SelectImage');
  },

  /**
   * Process a tab file and extract its metadata.
   */
  async processFile(path: string): Promise<Tab> {
    return await callBackend<Tab>('ProcessFile', path);
  },

  /**
   * Read text content from a PDF file.
   */
  async readPdf(path: string): Promise<string> {
    return await callBackend<string>('ReadPDF', path);
  },

  /**
   * Get the absolute path to the storage directory.
   */
  async getStorageDir(): Promise<string> {
    return await callBackend<string>('GetStorageDir');
  },

  /**
   * Get the absolute path to the covers directory.
   */
  async getCoversDir(): Promise<string> {
    return await callBackend<string>('GetCoversDir');
  },

  /**
   * Resolve a tab path to an absolute path.
   */
  async resolveTabPath(path: string, isManaged: boolean): Promise<string> {
    return await callBackend<string>('ResolveTabPath', path, isManaged);
  },

  /**
   * Resolve a cover path to an absolute path.
   */
  async resolveCoverPath(path: string): Promise<string> {
    return await callBackend<string>('ResolveCoverPath', path);
  },

  /**
   * Get the platform-correct URL for streaming a tab by ID.
   */
  async getTabContentURL(tabId: string): Promise<string> {
    return await callBackend<string>('GetTabContentURL', tabId);
  },

  /**
   * Get the platform-correct URL for loading a tab cover by ID.
   */
  async getCoverContentURL(tabId: string): Promise<string> {
    return await callBackend<string>('GetCoverContentURL', tabId);
  }
};
