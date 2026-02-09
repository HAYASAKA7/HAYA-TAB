package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"haya-tab/pkg/metadata"
	"haya-tab/pkg/store"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx   context.Context
	store *store.Store
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Init store
	cwd, _ := os.Getwd()
	dataPath := filepath.Join(cwd, "data", "tabs.json")
	a.store = store.NewStore(dataPath)
	if err := a.store.Load(); err != nil {
		fmt.Printf("Error loading store: %v", err)
	}
}

// GetTabs returns the list of tabs
func (a *App) GetTabs() []store.Tab {
	return a.store.Tabs
}

// ProcessFile takes a file path and returns a pre-filled Tab struct
func (a *App) ProcessFile(path string) store.Tab {
	meta := metadata.ParseFilename(path)
	ext := filepath.Ext(path)
	typeStr := "unknown"
	if ext == ".pdf" {
		typeStr = "pdf"
	} else if ext == ".gp" || ext == ".gp5" || ext == ".gpx" {
		typeStr = "gp"
	}

	return store.Tab{
		ID:       fmt.Sprintf("%d", time.Now().UnixNano()), // Simple ID
		Title:    meta.Title,
		Artist:   meta.Artist,
		Album:    meta.Album,
		FilePath: path,
		Type:     typeStr,
	}
}

// SaveTab saves the tab. copyFile determines if we import it to internal storage.
// The passed tab should have the user-confirmed Metadata.
func (a *App) SaveTab(tab store.Tab, shouldCopy bool) error {
	cwd, _ := os.Getwd()

	// 1. Handle File Copy
	if shouldCopy {
		ext := filepath.Ext(tab.FilePath)
		newFilename := tab.ID + ext
		destPath := filepath.Join(cwd, "storage", newFilename)

		src, err := os.Open(tab.FilePath)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return err
		}

		tab.FilePath = destPath
		tab.IsManaged = true
	} else {
		tab.IsManaged = false
	}

	// 2. Handle Cover (Async-ish, but we block here for simplicity or spawn goroutine)
	// We'll spawn a goroutine to not block UI, but we need to update the store later.
	// For now, let's block or just set a placeholder.
	// Let's try to fetch immediately.
	coverFilename := tab.ID + ".jpg"
	coverPath := filepath.Join(cwd, "covers", coverFilename)

	// If artist/album exists
	if tab.Artist != "" && tab.Album != "" {
		go func() {
			err := metadata.DownloadCover(tab.Artist, tab.Album, coverPath)
			if err == nil {
				// Update tab with cover path
				// We need to re-lock store. simpler to use a method on App that calls Store
				// But since we are inside App...
				// We'll cheat and just set it, then save.
				// Thread safety issue: we need to use the store's mutex.
				// Store has AddTab which locks.
				tab.CoverPath = coverPath
				a.store.AddTab(tab)
				wailsRuntime.EventsEmit(a.ctx, "tab-updated", tab) // Notify frontend
			} else {
				fmt.Printf("Failed to download cover: %v", err)
			}
		}()
	}

	// Save initial version (without cover maybe)
	return a.store.AddTab(tab)
}

// OpenTab opens the file using system default
func (a *App) OpenTab(id string) error {
	var targetTab store.Tab
	found := false
	for _, t := range a.store.Tabs {
		if t.ID == id {
			targetTab = t
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tab not found")
	}

	var cmd *exec.Cmd
	path := targetTab.FilePath

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // linux
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
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
