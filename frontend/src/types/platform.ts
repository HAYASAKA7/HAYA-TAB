export type RuntimeTarget = 'desktop' | 'ios' | 'android'
export type FormFactor = 'phone' | 'tablet' | 'desktop'

export interface RuntimeCapabilities {
  target: RuntimeTarget
  formFactor: FormFactor
  nativeTopLevelTabs: boolean
  webTopLevelTabs: boolean
  inProcessContent: boolean
  loopbackContent: boolean
  nativeFileImport: boolean
  safeAreaInsets: boolean
  folderWatcher: boolean
  customStoragePaths: boolean
  plugins: boolean
  webMIDI: boolean
  selfUpdate: boolean
}
