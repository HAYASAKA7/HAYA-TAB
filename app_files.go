package main

import (
	"encoding/base64"
	"haya-tab/pkg/store"
	"os"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// SelectFolder opens a folder selection dialog
func (a *App) SelectFolder() string {
	selection, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Destination Folder",
	})
	if err != nil {
		return ""
	}
	return selection
}

// SelectFiles opens a file dialog and returns the selected file paths
func (a *App) SelectFiles() []string {
	selection, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Tab Files",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Tabs (*.pdf;*.gp;*.gp3;*.gp4;*.gp5;*.gpx;*.xml;*.musicxml;*.mxl)", Pattern: "*.pdf;*.gp;*.gp3;*.gp4;*.gp5;*.gpx;*.xml;*.musicxml;*.mxl"},
		},
	})

	if err != nil {
		return nil
	}
	return selection
}

// SelectImage opens a file dialog for selecting images
func (a *App) SelectImage() string {
	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Image",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Images (*.jpg;*.png;*.jpeg;*.webp)", Pattern: "*.jpg;*.png;*.jpeg;*.webp"},
		},
	})

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

// TriggerSync delegates to SyncService for file synchronization
func (a *App) TriggerSync() (string, error) {
	return a.syncService.TriggerSync()
}
