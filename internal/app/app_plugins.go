package app

import "fmt"

// GetPlugins 返回所有已加载插件的信息
func (a *App) GetPlugins() []PluginInfo {
	if a.pluginManager == nil {
		return []PluginInfo{}
	}
	return a.pluginManager.GetPlugins()
}

// UpdatePluginConfig 更新插件的配置和启用状态
func (a *App) UpdatePluginConfig(id string, config map[string]string, enabled bool) error {
	if a.pluginManager == nil {
		return fmt.Errorf("plugin manager not initialized")
	}
	return a.pluginManager.UpdatePluginConfig(id, config, enabled)
}

// HasPlugins 检查系统是否加载了插件
func (a *App) HasPlugins() bool {
	if a.pluginManager == nil {
		return false
	}
	return a.pluginManager.HasPlugins()
}
