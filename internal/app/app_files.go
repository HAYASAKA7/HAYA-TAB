package app

import (
	"encoding/base64"
	"haya-tab/pkg/store"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SelectFolder opens a folder selection dialog
func (a *App) SelectFolder() string {
	dialog := application.Get().Dialog.OpenFile()
	dialog.SetTitle("Select Destination Folder").
		CanChooseDirectories(true).
		CanChooseFiles(false)

	selection, err := dialog.PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return selection
}

// SelectFiles opens a file dialog and returns the selected file paths
func (a *App) SelectFiles() []string {
	dialog := application.Get().Dialog.OpenFile()
	dialog.SetTitle("Select Tab Files").
		AddFilter("Tabs", "*.pdf;*.gp;*.gp3;*.gp4;*.gp5;*.gpx;*.xml;*.musicxml;*.mxl").
		CanChooseFiles(true).
		CanChooseDirectories(false)

	selection, err := dialog.PromptForMultipleSelection()
	if err != nil {
		return nil
	}
	return selection
}

// SelectImage opens a file dialog for selecting images
func (a *App) SelectImage() string {
	dialog := application.Get().Dialog.OpenFile()
	dialog.SetTitle("Select Image").
		AddFilter("Images", "*.jpg;*.png;*.jpeg;*.webp").
		CanChooseFiles(true).
		CanChooseDirectories(false)

	selection, err := dialog.PromptForSingleSelection()
	if err != nil {
		return ""
	}
	return selection
}

// GetCover returns the base64 encoded image
func (a *App) GetCover(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

// fetchCoverAsync delegates to SyncService for async cover download
func (a *App) fetchCoverAsync(tab store.Tab) {
	a.syncService.FetchCoverAsync(tab)
}

// TriggerSync delegates to SyncService for file synchronization.
// It also notifies the PluginManager that a new sync run is starting so
// plugins can reset per-run counters (e.g. AI request limits).
func (a *App) TriggerSync() (string, error) {
	if a.pluginManager != nil {
		a.pluginManager.StartSyncRun()
	}
	return a.syncService.TriggerSync()
}
