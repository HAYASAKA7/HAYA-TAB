package app

import (
	"encoding/json"
	"fmt"
	apperrors "haya-tab/pkg/errors"
	syncpkg "haya-tab/pkg/sync"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// FileCacheMaxAge is the Cache-Control max-age for streamed tab files (1 hour).
	FileCacheMaxAge = 3600
	// CoverCacheMaxAge is the Cache-Control max-age for cover images (24 hours).
	CoverCacheMaxAge = 86400
)

// StartFileServer starts a local HTTP server to serve files
func (a *App) StartFileServer() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to bind to random port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	handler := &FileHandler{app: a}
	mux.Handle("/", handler)

	fmt.Printf("[FileServer] Listening on http://127.0.0.1:%d\n", port)

	go func() {
		if err := http.Serve(listener, mux); err != nil {
			fmt.Printf("FileServer error: %v\n", err)
		}
	}()

	return port, nil
}

// FileHandler handles HTTP requests for streaming files
type FileHandler struct {
	app *App
}

// NewFileHandler creates a new file handler
func NewFileHandler(app *App) *FileHandler {
	return &FileHandler{app: app}
}

// ServeHTTP implements http.Handler for streaming files
func (h *FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[FileHandler] ServeHTTP called: %s %s\n", r.Method, r.URL.String())

	// Enable CORS for local development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	// Only log api calls to avoid noise
	if strings.HasPrefix(path, "/api/") {
		fmt.Printf("[FileHandler] Request: %s\n", path)
	}

	// Handle /api/file/{id} - stream tab file content
	if strings.HasPrefix(path, "/api/file/") {
		h.serveTabFile(w, r, strings.TrimPrefix(path, "/api/file/"))
		return
	}

	// Handle /api/cover/{id} - stream cover image
	if strings.HasPrefix(path, "/api/cover/") {
		id := strings.TrimPrefix(path, "/api/cover/")
		// Strip query parameters if present
		if idx := strings.Index(id, "?"); idx != -1 {
			id = id[:idx]
		}
		h.serveCoverFile(w, r, id)
		return
	}

	// Handle /api/cloud-stream/{id} - stream cloud tab file via WebDAV proxy
	if strings.HasPrefix(path, "/api/cloud-stream/") {
		id := strings.TrimPrefix(path, "/api/cloud-stream/")
		// Strip query parameters if present
		if idx := strings.Index(id, "?"); idx != -1 {
			id = id[:idx]
		}
		h.serveCloudFile(w, r, id)
		return
	}

	// Not found
	http.NotFound(w, r)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   apperrors.ToPayload(err),
	})
}

func (h *FileHandler) serveTabFile(w http.ResponseWriter, r *http.Request, id string) {
	fmt.Printf("[ServeTabFile] Request for ID: %s\n", id)
	if h.app == nil || h.app.store == nil {
		fmt.Println("[ServeTabFile] Store is nil")
		writeAPIError(w, http.StatusServiceUnavailable, apperrors.NewAppError("errors.serviceUnavailable", "service unavailable", nil, nil))
		return
	}

	tab, err := h.app.store.GetTab(id)
	if err != nil {
		fmt.Printf("[ServeTabFile] Error getting tab %s: %v\n", id, err)
		writeAPIError(w, http.StatusBadRequest, apperrors.WrapError(err, apperrors.ErrCodeDatabase, map[string]interface{}{"operation": "get_tab"}))
		return
	}
	if tab == nil {
		fmt.Printf("[ServeTabFile] Tab not found for ID: %s\n", id)
		writeAPIError(w, http.StatusBadRequest, apperrors.TabNotFoundError(id))
		return
	}

	// If it's a cloud tab, use the cloud stream handler
	if tab.IsCloud {
		fmt.Printf("[ServeTabFile] Tab %s is a cloud tab, using cloud stream\n", id)
		h.serveCloudFile(w, r, id)
		return
	}

	fullPath := h.app.ResolveTabPath(tab.FilePath, tab.IsManaged)
	fmt.Printf("[ServeTabFile] Found tab: %s, Path: %s\n", tab.Title, fullPath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fmt.Printf("[ServeTabFile] File not found: %s\n", fullPath)
		writeAPIError(w, http.StatusNotFound, apperrors.FileNotFoundError(fullPath, err))
		return
	}

	// Open the file
	file, err := os.Open(fullPath)
	if err != nil {
		fmt.Printf("[ServeTabFile] Failed to open file %s: %v\n", fullPath, err)
		writeAPIError(w, http.StatusInternalServerError, apperrors.FileAccessError(fullPath, err))
		return
	}
	defer file.Close()

	// Get file info for content-length
	stat, err := file.Stat()
	if err != nil {
		fmt.Printf("[ServeTabFile] Failed to stat file: %v\n", err)
		writeAPIError(w, http.StatusInternalServerError, apperrors.FileAccessError(fullPath, err))
		return
	}

	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".gp", ".gp3", ".gp4", ".gp5", ".gpx":
		contentType = "application/x-guitar-pro"
	case ".xml", ".musicxml":
		contentType = "application/vnd.recordare.musicxml+xml"
	case ".mxl":
		contentType = "application/vnd.recordare.musicxml"
	}

	// Set headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(tab.FilePath)))
	w.Header().Set("Cache-Control", fmt.Sprintf("private, max-age=%d", FileCacheMaxAge))

	// Stream the file
	io.Copy(w, file)
}

func (h *FileHandler) serveCoverFile(w http.ResponseWriter, r *http.Request, id string) {
	fmt.Printf("[ServeCoverFile] Request for ID: %s\n", id)
	if h.app == nil || h.app.store == nil {
		fmt.Println("[ServeCoverFile] Store is nil")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	tab, err := h.app.store.GetTab(id)
	if err != nil || tab == nil {
		fmt.Printf("[ServeCoverFile] Tab not found for ID: %s, err: %v\n", id, err)
		http.Error(w, "Tab not found", http.StatusNotFound)
		return
	}

	if tab.CoverPath == "" {
		fmt.Printf("[ServeCoverFile] No cover path for tab: %s\n", id)
		http.Error(w, "No cover available", http.StatusNotFound)
		return
	}

	fullCoverPath := h.app.ResolveCoverPath(tab.CoverPath)
	fmt.Printf("[ServeCoverFile] Opening cover: %s\n", fullCoverPath)

	// Check if file exists
	if _, err := os.Stat(fullCoverPath); os.IsNotExist(err) {
		fmt.Printf("[ServeCoverFile] Cover not found: %s\n", fullCoverPath)
		http.Error(w, "Cover not found", http.StatusNotFound)
		return
	}

	// Open the cover file
	file, err := os.Open(fullCoverPath)
	if err != nil {
		fmt.Printf("[ServeCoverFile] Failed to open cover: %v\n", err)
		http.Error(w, "Cover not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Cannot read cover", http.StatusInternalServerError)
		return
	}

	// Determine content type
	ext := strings.ToLower(filepath.Ext(fullCoverPath))
	contentType := "image/jpeg"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".gif":
		contentType = "image/gif"
	}

	// Set headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", CoverCacheMaxAge)) // Cache covers for 24 hours

	// Stream the file
	io.Copy(w, file)
}

func (h *FileHandler) serveCloudFile(w http.ResponseWriter, r *http.Request, id string) {
	fmt.Printf("[ServeCloudFile] Request for ID: %s\n", id)
	if h.app == nil || h.app.store == nil {
		fmt.Println("[ServeCloudFile] Store is nil")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	tab, err := h.app.store.GetTab(id)
	if err != nil {
		fmt.Printf("[ServeCloudFile] Error getting tab %s: %v\n", id, err)
		http.Error(w, "Tab not found", http.StatusBadRequest)
		return
	}
	if tab == nil {
		fmt.Printf("[ServeCloudFile] Tab not found for ID: %s\n", id)
		http.Error(w, "Tab not found", http.StatusBadRequest)
		return
	}

	// DEBUG: Print tab details
	fmt.Printf("[ServeCloudFile] Tab details: ID=%s, VolumeID='%s', FilePath='%s', IsCloud=%v\n",
		tab.ID, tab.VolumeID, tab.FilePath, tab.IsCloud)

	if !tab.IsCloud {
		fmt.Printf("[ServeCloudFile] Tab %s is not a cloud tab, redirecting to local\n", id)
		h.serveTabFile(w, r, id)
		return
	}

	// Get WebDAV settings
	settings := h.app.store.GetSettings()
	if !settings.WebDAVEnabled || settings.WebDAVURL == "" {
		fmt.Println("[ServeCloudFile] WebDAV not enabled")
		http.Error(w, "WebDAV not configured", http.StatusServiceUnavailable)
		return
	}

	// Create WebDAV client with browser-like User-Agent
	baseURL := strings.TrimRight(settings.WebDAVURL, "/")
	client := syncpkg.NewWebDAVClient(baseURL, settings.WebDAVUser, settings.WebDAVPassword)

	// Resolve the full WebDAV path using volume system
	fullPath, err := syncpkg.ResolveFilePath(h.app.store, tab.VolumeID, tab.FilePath)
	if err != nil {
		fmt.Printf("[ServeCloudFile] Failed to resolve file path: %v\n", err)
		http.Error(w, "Failed to resolve file path: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[ServeCloudFile] Streaming from WebDAV: %s (resolved from volume %s)\n", fullPath, tab.VolumeID)

	// Get file info for content-length
	var fileSize int64
	fileInfo, err := client.GetFileInfo(fullPath)
	if err != nil {
		fmt.Printf("[ServeCloudFile] Failed to get file info: %v\n", err)
		// Continue without file size - not fatal
	} else {
		fileSize = fileInfo.Size()
	}

	// Create context that cancels when client disconnects
	ctx := r.Context()

	// Use gowebdav's ReadStream which handles auth properly for Baidu/Jianguoyun etc.
	stream, err := client.ReadStream(fullPath)
	if err != nil {
		fmt.Printf("[ServeCloudFile] Failed to open WebDAV stream: %v\n", err)
		errStr := err.Error()
		if strings.Contains(errStr, "403") || strings.Contains(strings.ToLower(errStr), "forbidden") {
			http.Error(w, "Access denied (403): Server rejected the request", http.StatusForbidden)
			return
		}
		if strings.Contains(errStr, "404") || strings.Contains(strings.ToLower(errStr), "not found") {
			http.Error(w, "File not found on cloud storage", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to stream file: "+errStr, http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(tab.FilePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".gp", ".gp3", ".gp4", ".gp5", ".gpx":
		contentType = "application/x-guitar-pro"
	case ".xml", ".musicxml":
		contentType = "application/vnd.recordare.musicxml+xml"
	case ".mxl":
		contentType = "application/vnd.recordare.musicxml"
	}

	// Set response headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(tab.FilePath)))
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("Accept-Ranges", "bytes")

	if fileSize > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
	}

	// Stream the file with context awareness for cancellation
	done := make(chan struct{})
	var written int64
	var copyErr error

	go func() {
		written, copyErr = io.Copy(w, stream)
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Client disconnected - close stream to stop the goroutine
		stream.Close()
		fmt.Printf("[ServeCloudFile] Client disconnected after streaming started\n")
		return
	case <-done:
		if copyErr != nil {
			fmt.Printf("[ServeCloudFile] Error streaming file: %v\n", copyErr)
			return
		}
		fmt.Printf("[ServeCloudFile] Streamed %d bytes successfully\n", written)
	}
}
