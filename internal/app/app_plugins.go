package app

import apperrors "haya-tab/pkg/errors"

// GetPlugins returns information for all loaded plugins.
func (a *App) GetPlugins() []PluginInfo {
	if a.pluginManager == nil {
		return []PluginInfo{}
	}
	return a.pluginManager.GetPlugins()
}

// UpdatePluginConfig updates a plugin's configuration and enabled state.
func (a *App) UpdatePluginConfig(id string, config map[string]string, enabled bool) error {
	if a.pluginManager == nil {
		return apperrors.PluginManagerNotInitializedError()
	}
	return a.pluginManager.UpdatePluginConfig(id, config, enabled)
}

// HasPlugins checks whether any plugins are loaded.
func (a *App) HasPlugins() bool {
	if a.pluginManager == nil {
		return false
	}
	return a.pluginManager.HasPlugins()
}
