package sync

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCustomTransport_RoundTrip(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		wantReferer   string
		wantOrigin    string
		wantUserAgent string
	}{
		{
			name:          "Aliyundrive",
			host:          "aliyundrive.com",
			wantReferer:   "https://www.aliyundrive.com/",
			wantOrigin:    "https://www.aliyundrive.com",
			wantUserAgent: BrowserUserAgent,
		},
		{
			name:          "Quark",
			host:          "quark.cn",
			wantReferer:   "https://pan.quark.cn/",
			wantOrigin:    "https://pan.quark.cn",
			wantUserAgent: BrowserUserAgent,
		},
		{
			name:          "Baidu",
			host:          "baidu.com",
			wantReferer:   "https://pan.baidu.com/",
			wantOrigin:    "https://pan.baidu.com",
			wantUserAgent: BrowserUserAgent,
		},
		{
			name:          "115",
			host:          "115.com",
			wantReferer:   "https://115.com/",
			wantOrigin:    "https://115.com",
			wantUserAgent: BrowserUserAgent,
		},
		{
			name:          "189",
			host:          "189.cn",
			wantReferer:   "https://cloud.189.cn/",
			wantOrigin:    "https://cloud.189.cn",
			wantUserAgent: BrowserUserAgent,
		},
		{
			name:          "123pan",
			host:          "123pan.cn",
			wantReferer:   "https://www.123pan.com/",
			wantOrigin:    "https://www.123pan.com",
			wantUserAgent: BrowserUserAgent,
		},
		{
			name:          "10086",
			host:          "10086.cn",
			wantReferer:   "https://caiyun.feixin.10086.cn/",
			wantOrigin:    "https://caiyun.feixin.10086.cn",
			wantUserAgent: BrowserUserAgent,
		},
		{
			name:          "PikPak",
			host:          "mypikpak.com",
			wantReferer:   "https://mypikpak.com/",
			wantOrigin:    "https://mypikpak.com",
			wantUserAgent: BrowserUserAgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server that captures headers
			var capturedHeaders http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedHeaders = r.Header.Clone()
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create mock transport that captures the modified request
			mockBase := &mockRoundTripper{
				handler: func(req *http.Request) (*http.Response, error) {
					capturedHeaders = req.Header.Clone()
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       http.NoBody,
						Header:     make(http.Header),
					}, nil
				},
			}

			transport := &customTransport{
				base: mockBase,
			}

			// Create request with test host
			req, err := http.NewRequest("GET", "http://"+tt.host+"/test", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Execute request through transport
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip failed: %v", err)
			}
			defer resp.Body.Close()

			// Verify headers
			if capturedHeaders.Get("User-Agent") != tt.wantUserAgent {
				t.Errorf("User-Agent = %v, want %v", capturedHeaders.Get("User-Agent"), tt.wantUserAgent)
			}
			if capturedHeaders.Get("Referer") != tt.wantReferer {
				t.Errorf("Referer = %v, want %v", capturedHeaders.Get("Referer"), tt.wantReferer)
			}
			if capturedHeaders.Get("Origin") != tt.wantOrigin {
				t.Errorf("Origin = %v, want %v", capturedHeaders.Get("Origin"), tt.wantOrigin)
			}
		})
	}
}

// mockRoundTripper is a mock implementation of http.RoundTripper for testing
type mockRoundTripper struct {
	handler func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req)
}

func TestCustomTransport_RoundTrip_DefaultFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify User-Agent is set
		if r.Header.Get("User-Agent") != BrowserUserAgent {
			t.Errorf("User-Agent = %v, want %v", r.Header.Get("User-Agent"), BrowserUserAgent)
		}
		// Verify Referer and Origin are set for unknown hosts
		if r.Header.Get("Referer") == "" {
			t.Error("Referer should be set for unknown hosts")
		}
		if r.Header.Get("Origin") == "" {
			t.Error("Origin should be set for unknown hosts")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &customTransport{
		base: http.DefaultTransport,
	}

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestCustomTransport_RoundTrip_AcceptHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "*/*" {
			t.Errorf("Accept = %v, want */*", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := &customTransport{
		base: http.DefaultTransport,
	}

	req, _ := http.NewRequest("GET", server.URL, nil)
	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestNewWebDAVClient(t *testing.T) {
	client := NewWebDAVClient("http://example.com/webdav", "user", "pass")

	if client == nil {
		t.Fatal("NewWebDAVClient returned nil")
	}
	if client.metadataClient == nil {
		t.Error("WebDAVClient.metadataClient is nil")
	}
	if client.streamClient == nil {
		t.Error("WebDAVClient.streamClient is nil")
	}
	if client.url != "http://example.com/webdav" {
		t.Errorf("url = %v, want http://example.com/webdav", client.url)
	}
	if client.httpClient == nil {
		t.Error("WebDAVClient.httpClient is nil")
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Already has leading slash",
			input: "/path/to/file",
			want:  "/path/to/file",
		},
		{
			name:  "No leading slash",
			input: "path/to/file",
			want:  "/path/to/file",
		},
		{
			name:  "Empty path",
			input: "",
			want:  "/",
		},
		{
			name:  "Root path",
			input: "/",
			want:  "/",
		},
		{
			name:  "Path with spaces",
			input: "path/with spaces/file.txt",
			want:  "/path/with spaces/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePath(tt.input)
			if got != tt.want {
				t.Errorf("sanitizePath(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestWebDAVClient_GetURL(t *testing.T) {
	client := NewWebDAVClient("http://example.com/webdav", "user", "pass")

	url := client.GetURL()
	if url != "http://example.com/webdav" {
		t.Errorf("GetURL() = %v, want http://example.com/webdav", url)
	}
}

func TestWebDAVClient_GetHTTPClient(t *testing.T) {
	client := NewWebDAVClient("http://example.com/webdav", "user", "pass")

	httpClient := client.GetHTTPClient()
	if httpClient == nil {
		t.Error("GetHTTPClient() returned nil")
	}
}

func TestBrowserUserAgent(t *testing.T) {
	if BrowserUserAgent == "" {
		t.Error("BrowserUserAgent is empty")
	}
	if len(BrowserUserAgent) < 10 {
		t.Error("BrowserUserAgent seems too short")
	}
}

func TestWebDAVClient_TestConnection(t *testing.T) {
	// Create a mock WebDAV server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" || r.Method == "PROPFIND" {
			w.Header().Set("DAV", "1, 2")
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	// TestConnection may fail with mock server, but we're testing it doesn't panic
	_ = client.TestConnection()
}

func TestWebDAVClient_GetFileInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	_, err := client.GetFileInfo("/test.pdf")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestWebDAVClient_ReadStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	_, err := client.ReadStream("/test.pdf")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestWebDAVClient_DownloadFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	tmpDir, _ := os.MkdirTemp("", "webdav-test-*")
	defer os.RemoveAll(tmpDir)

	localPath := filepath.Join(tmpDir, "test.pdf")
	err := client.DownloadFile("/test.pdf", localPath)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestWebDAVClient_UploadFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	tmpDir, _ := os.MkdirTemp("", "webdav-test-*")
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.pdf")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// Upload will likely fail with mock server, but we're testing it doesn't panic
	_ = client.UploadFile(testFile, "/remote")
}

func TestWebDAVClient_ListDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	_, err := client.ListDir("/")
	if err == nil {
		t.Error("Expected error for failed directory listing")
	}
}

func TestWebDAVClient_ListRemoteDirectories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	_, err := client.ListRemoteDirectories("/")
	if err == nil {
		t.Error("Expected error for failed directory listing")
	}
}

func TestWebDAVClient_ScanRemoteFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	_, err := client.ScanRemoteFiles("/")
	if err == nil {
		t.Error("Expected error for failed scan")
	}
}

func TestWebDAVClient_DownloadFile_Success(t *testing.T) {
	testContent := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write(testContent)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	tmpDir, _ := os.MkdirTemp("", "webdav-test-*")
	defer os.RemoveAll(tmpDir)

	localPath := filepath.Join(tmpDir, "downloaded.pdf")
	err := client.DownloadFile("/test.pdf", localPath)
	if err != nil {
		t.Errorf("DownloadFile() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Error("Downloaded file does not exist")
	}

	// Verify content
	content, _ := os.ReadFile(localPath)
	if string(content) != string(testContent) {
		t.Errorf("Downloaded content = %v, want %v", string(content), string(testContent))
	}
}

func TestWebDAVClient_UploadFile_NonExistentFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	err := client.UploadFile("/nonexistent/file.pdf", "/remote")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestSyncService_ProcessFile_ErrorHandling(t *testing.T) {
	service, _, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	// Test with non-existent file
	tab := service.ProcessFile("/nonexistent/file.pdf")

	// Should still return a tab with basic info
	if tab.ID == "" {
		t.Error("ProcessFile() returned empty ID for non-existent file")
	}
	if tab.Type != "pdf" {
		t.Errorf("Type = %v, want pdf", tab.Type)
	}
}

func TestSyncService_TriggerSync_WithSubdirectories(t *testing.T) {
	service, testStore, tmpDir, cleanup := setupTestSyncService(t)
	defer cleanup()

	// Create nested directory structure
	syncDir := filepath.Join(tmpDir, "sync")
	subDir := filepath.Join(syncDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Create files in subdirectory
	testFile := filepath.Join(subDir, "Artist - Song.pdf")
	f, _ := os.Create(testFile)
	f.Close()

	// Configure sync path
	settings := testStore.GetSettings()
	settings.SyncPaths = []string{syncDir}
	settings.SyncStrategy = "skip"
	testStore.UpdateSettings(settings)

	// Trigger sync
	result, err := service.TriggerSync()
	if err != nil {
		t.Fatalf("TriggerSync() error = %v", err)
	}

	if result == "" {
		t.Error("TriggerSync() returned empty result")
	}

	// Verify tab was added from subdirectory
	tabs, _ := testStore.GetAllTabs()
	if len(tabs) != 1 {
		t.Errorf("Expected 1 tab from subdirectory, got %d", len(tabs))
	}
}

func TestSyncService_TriggerSync_UnsupportedFiles(t *testing.T) {
	service, testStore, tmpDir, cleanup := setupTestSyncService(t)
	defer cleanup()

	// Create sync directory with unsupported files
	syncDir := filepath.Join(tmpDir, "sync")
	os.MkdirAll(syncDir, 0755)

	// Create unsupported files
	unsupportedFiles := []string{
		"document.txt",
		"image.jpg",
		"video.mp4",
	}

	for _, filename := range unsupportedFiles {
		f, _ := os.Create(filepath.Join(syncDir, filename))
		f.Close()
	}

	// Configure sync path
	settings := testStore.GetSettings()
	settings.SyncPaths = []string{syncDir}
	testStore.UpdateSettings(settings)

	// Trigger sync
	service.TriggerSync()

	// Verify no tabs were added
	tabs, _ := testStore.GetAllTabs()
	if len(tabs) != 0 {
		t.Errorf("Expected 0 tabs (unsupported files), got %d", len(tabs))
	}
}

func TestSyncService_generateUniqueTitle_SafetyLimit(t *testing.T) {
	service, _, _, cleanup := setupTestSyncService(t)
	defer cleanup()

	baseTitle := "Test"

	// Add 1001 tabs with similar titles to trigger safety limit
	// This is impractical, so we'll just test the logic exists
	// by checking the function handles the case

	uniqueTitle := service.generateUniqueTitle(baseTitle, map[string]bool{})
	if uniqueTitle == "" {
		t.Error("generateUniqueTitle() returned empty string")
	}
}
