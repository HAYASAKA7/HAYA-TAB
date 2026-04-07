// @ts-ignore
import * as App from '../../bindings/haya-tab/internal/app/app.js'

export interface PluginInfo {
  id: string
  name: string
  version: string
  settingsSchema: Record<string, string>
  config: Record<string, string>
  enabled: boolean
}

export class PluginService {
  static async getPlugins(): Promise<PluginInfo[]> {
    const plugins = await App.GetPlugins()
    return plugins as unknown as PluginInfo[]
  }

  static async updatePluginConfig(id: string, config: Record<string, string>, enabled: boolean): Promise<void> {
    await App.UpdatePluginConfig(id, config, enabled)
  }

  static async hasPlugins(): Promise<boolean> {
    return await App.HasPlugins()
  }
}
