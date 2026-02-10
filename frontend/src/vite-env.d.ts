/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

interface Window {
  go: {
    main: {
      App: {
        GetTabs(): Promise<import('./types').Tab[]>
        GetTabsPaginated(categoryId: string, page: number, pageSize: number): Promise<import('./types').TabsResponse>
        GetCategories(): Promise<import('./types').Category[]>
        GetSettings(): Promise<import('./types').Settings>
        SaveSettings(settings: import('./types').Settings): Promise<void>
        AddCategory(category: import('./types').Category): Promise<void>
        DeleteCategory(id: string): Promise<void>
        MoveCategory(id: string, newParentId: string): Promise<void>
        SaveTab(tab: import('./types').Tab, shouldCopy: boolean): Promise<void>
        UpdateTab(tab: import('./types').Tab): Promise<void>
        DeleteTab(id: string): Promise<void>
        MoveTab(tabId: string, categoryId: string): Promise<void>
        BatchDeleteTabs(ids: string[]): Promise<number>
        BatchMoveTabs(ids: string[], categoryId: string): Promise<number>
        OpenTab(id: string): Promise<void>
        ExportTab(id: string, destFolder: string): Promise<void>
        ProcessFile(path: string): Promise<import('./types').Tab>
        SelectFiles(): Promise<string[]>
        SelectFolder(): Promise<string>
        SelectImage(): Promise<string>
        TriggerSync(): Promise<string>
        GetCover(path: string): Promise<string>
        GetTabContent(id: string): Promise<string>
      }
    }
  }
  runtime: {
    EventsOn(event: string, callback: (...args: any[]) => void): void
    EventsOff(event: string): void
    EventsEmit(event: string, ...args: any[]): void
  }
}
