package app

import "haya-tab/internal/platform"

func (a *App) GetRuntimeCapabilities(viewportWidth int) platform.Capabilities {
	return platform.CapabilitiesFor(platform.CurrentTarget(), viewportWidth)
}
