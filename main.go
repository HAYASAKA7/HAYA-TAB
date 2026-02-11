package main

import (
	"embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// StartFileServer starts a local HTTP server to serve files
func StartFileServer(app *App) (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to bind to random port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	handler := &FileHandler{app: app}
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
		h.serveCoverFile(w, r, strings.TrimPrefix(path, "/api/cover/"))
		return
	}

	// Not found
	http.NotFound(w, r)
}

func (h *FileHandler) serveTabFile(w http.ResponseWriter, r *http.Request, id string) {
	fmt.Printf("[ServeTabFile] Request for ID: %s\n", id)
	if h.app == nil || h.app.store == nil {
		fmt.Println("[ServeTabFile] Store is nil")
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	tab, err := h.app.store.GetTab(id)
	if err != nil {
		fmt.Printf("[ServeTabFile] Error getting tab %s: %v\n", id, err)
		http.Error(w, "Tab not found", http.StatusBadRequest)
		return
	}
	if tab == nil {
		fmt.Printf("[ServeTabFile] Tab not found for ID: %s\n", id)
		http.Error(w, "Tab not found", http.StatusBadRequest)
		return
	}

	fmt.Printf("[ServeTabFile] Found tab: %s, Path: %s\n", tab.Title, tab.FilePath)

	// Open the file
	file, err := os.Open(tab.FilePath)
	if err != nil {
		fmt.Printf("[ServeTabFile] Failed to open file %s: %v\n", tab.FilePath, err)
		http.Error(w, "File not found", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info for content-length
	stat, err := file.Stat()
	if err != nil {
		fmt.Printf("[ServeTabFile] Failed to stat file: %v\n", err)
		http.Error(w, "Cannot read file", http.StatusInternalServerError)
		return
	}

	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(tab.FilePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".gp", ".gp5", ".gpx":
		contentType = "application/x-guitar-pro"
	}

	// Set headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(tab.FilePath)))
	w.Header().Set("Cache-Control", "private, max-age=3600")

	// Stream the file
	io.Copy(w, file)
}

func (h *FileHandler) serveCoverFile(w http.ResponseWriter, r *http.Request, id string) {
	if h.app == nil || h.app.store == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	tab, err := h.app.store.GetTab(id)
	if err != nil || tab == nil {
		http.Error(w, "Tab not found", http.StatusNotFound)
		return
	}

	if tab.CoverPath == "" {
		http.Error(w, "No cover available", http.StatusNotFound)
		return
	}

	// Open the cover file
	file, err := os.Open(tab.CoverPath)
	if err != nil {
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
	ext := strings.ToLower(filepath.Ext(tab.CoverPath))
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
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache covers for 24 hours

	// Stream the file
	io.Copy(w, file)
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Start local file server
	port, err := StartFileServer(app)
	if err != nil {
		println("Error starting file server:", err.Error())
		// In a GUI app, we might want to show a dialog, but main() runs before wails.Run,
		// so we can't use wails runtime dialogs yet. Standard output is best effort here.
		return
	}
	app.SetFileServerPort(port)

	// Create file handler for streaming
	fileHandler := NewFileHandler(app)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "HAYA-TAB",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: fileHandler,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
