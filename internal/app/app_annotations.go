package app

import (
	"fmt"
	"path"
	"strings"

	"haya-tab/pkg/store"
	syncpkg "haya-tab/pkg/sync"
)

const annotationDirectoryName = "annotations"

// getAnnotationRemotePath builds the canonical WebDAV path for a page annotation.
// Layout: <volume_mount>/haya-metadata/annotations/<relative_file_path>.p<page>.json
// Returns false when the tab/page cannot be mapped to a safe cloud-relative path.
func (a *App) getAnnotationRemotePath(tab *store.Tab, pageNumber int) (string, bool) {
	if tab == nil || tab.VolumeID == "" || pageNumber < 1 {
		return "", false
	}

	relativePath := strings.TrimSpace(tab.CloudPath)
	if relativePath == "" && tab.IsCloud {
		relativePath = strings.TrimSpace(tab.FilePath)
	}
	if relativePath == "" {
		return "", false
	}

	relativePath = strings.TrimPrefix(relativePath, "/")
	relativePath = strings.ReplaceAll(relativePath, "\\", "/")
	relativePath = path.Clean(relativePath)
	if relativePath == "." || relativePath == "" || relativePath == ".." || strings.HasPrefix(relativePath, "../") {
		return "", false
	}

	volume, err := a.store.GetVolume(tab.VolumeID)
	if err != nil || volume == nil || !volume.IsAvailable {
		return "", false
	}

	filename := fmt.Sprintf("%s.p%d.json", relativePath, pageNumber)
	remotePath := path.Join(volume.MountPath, syncpkg.MetadataDirectoryName, annotationDirectoryName, filename)
	return remotePath, true
}

// isWebDAVNotFoundError identifies common "missing file" errors returned by WebDAV backends.
// These errors are treated as normal in lazy-fetch flows.
func isWebDAVNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "not found") || strings.Contains(msg, "no such")
}

// syncAnnotationToWebDAV performs best-effort upload of page annotation JSON to WebDAV.
// It is intentionally non-blocking and logs failures without interrupting local saves.
func (a *App) syncAnnotationToWebDAV(tabID string, pageNumber int, jsonData string) {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return
	}

	tab, err := a.store.GetTab(tabID)
	if err != nil || tab == nil {
		return
	}

	remotePath, ok := a.getAnnotationRemotePath(tab, pageNumber)
	if !ok {
		return
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	if err := client.WriteBytes(remotePath, []byte(jsonData)); err != nil {
		a.logger.Error("Failed to sync annotation to WebDAV (tab=%s, page=%d, path=%s): %v", tabID, pageNumber, remotePath, err)
		return
	}

	a.logger.Info("Synced annotation to WebDAV (tab=%s, page=%d)", tabID, pageNumber)
}

// fetchAnnotationFromWebDAV lazily retrieves page annotation JSON from WebDAV when local data is empty.
// On success, it also writes the data back to local SQLite cache for subsequent fast reads.
func (a *App) fetchAnnotationFromWebDAV(tabID string, pageNumber int) (string, bool) {
	settings := a.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		return "", false
	}

	tab, err := a.store.GetTab(tabID)
	if err != nil || tab == nil {
		return "", false
	}

	remotePath, ok := a.getAnnotationRemotePath(tab, pageNumber)
	if !ok {
		return "", false
	}

	client := syncpkg.NewWebDAVClient(
		strings.TrimRight(settings.WebDAVURL, "/"),
		settings.WebDAVUser,
		settings.WebDAVPassword,
	)

	data, err := client.ReadBytes(remotePath)
	if err != nil {
		if !isWebDAVNotFoundError(err) {
			a.logger.Warning("Failed to fetch annotation from WebDAV (tab=%s, page=%d, path=%s): %v", tabID, pageNumber, remotePath, err)
		}
		return "", false
	}

	jsonData := strings.TrimSpace(string(data))
	if jsonData == "" {
		jsonData = "[]"
	}

	if err := a.store.SaveTabAnnotation(tabID, pageNumber, jsonData); err != nil {
		a.logger.Warning("Failed to persist fetched annotation locally (tab=%s, page=%d): %v", tabID, pageNumber, err)
	}

	return jsonData, true
}
