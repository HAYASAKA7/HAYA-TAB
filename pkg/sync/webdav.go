package sync

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"haya-tab/pkg/store"

	"github.com/studio-b12/gowebdav"
)

// BrowserUserAgent is a Chrome-like User-Agent to avoid 403 blocks from WebDAV servers
const BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

const (
	// DefaultMetadataTimeout is the HTTP request timeout for WebDAV metadata operations.
	DefaultMetadataTimeout = 30 * time.Second
	// MaxDirectoryScanDepth is the maximum recursion depth when scanning remote directories.
	MaxDirectoryScanDepth = 10
	// MaxIdleConns is the maximum number of idle connections in the HTTP transport.
	MaxIdleConns = 10
	// IdleConnTimeout is the timeout for idle connections before they are closed.
	IdleConnTimeout = 90 * time.Second
	// MaxIdleConnsPerHost is the maximum number of idle connections per host.
	MaxIdleConnsPerHost = 5
	// MaxConnsPerHost is the strict concurrency limit per host to avoid overwhelming WebDAV servers.
	MaxConnsPerHost = 5
)

// customTransport wraps http.Transport to inject headers on every request
type customTransport struct {
	base http.RoundTripper
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid modifying the original request concurrently
	clone := req.Clone(req.Context())

	// Set browser-like User-Agent
	clone.Header.Set("User-Agent", BrowserUserAgent)
	// Add Accept header for better compatibility
	if clone.Header.Get("Accept") == "" {
		clone.Header.Set("Accept", "*/*")
	}

	if clone.URL != nil {
		host := strings.ToLower(clone.URL.Host)

		// Bypass Anti-Hotlinking based on domain
		switch {
		case strings.Contains(host, "aliyundrive") || strings.Contains(host, "alicloudccp"):
			clone.Header.Set("Referer", "https://www.aliyundrive.com/")
			clone.Header.Set("Origin", "https://www.aliyundrive.com")
		case strings.Contains(host, "quark"):
			clone.Header.Set("Referer", "https://pan.quark.cn/")
			clone.Header.Set("Origin", "https://pan.quark.cn")
		case strings.Contains(host, "baidu.com") || strings.Contains(host, "bdimg.com") || strings.Contains(host, "baidupcs.com"):
			clone.Header.Set("Referer", "https://pan.baidu.com/")
			clone.Header.Set("Origin", "https://pan.baidu.com")
		case strings.Contains(host, "115.com") || strings.Contains(host, "115.net"):
			clone.Header.Set("Referer", "https://115.com/")
			clone.Header.Set("Origin", "https://115.com")
		case strings.Contains(host, "189.cn"):
			clone.Header.Set("Referer", "https://cloud.189.cn/")
			clone.Header.Set("Origin", "https://cloud.189.cn")
		case strings.Contains(host, "123pan.cn") || strings.Contains(host, "123pan.com"):
			clone.Header.Set("Referer", "https://www.123pan.com/")
			clone.Header.Set("Origin", "https://www.123pan.com")
		case strings.Contains(host, "10086.cn"):
			clone.Header.Set("Referer", "https://caiyun.feixin.10086.cn/")
			clone.Header.Set("Origin", "https://caiyun.feixin.10086.cn")
		case strings.Contains(host, "mypikpak.com"):
			clone.Header.Set("Referer", "https://mypikpak.com/")
			clone.Header.Set("Origin", "https://mypikpak.com")
		default:
			// Fallback for global drives and initial WebDAV requests
			// Escape Redirect URLs to prevent 400 Bad Request on strict CDNs due to unescaped spaces
			if clone.URL.Path != "" {
				clone.URL.RawPath = strings.ReplaceAll(url.PathEscape(clone.URL.Path), "%2F", "/")
			}

			if clone.Header.Get("Referer") == "" {
				clone.Header.Set("Referer", clone.URL.Scheme+"://"+clone.URL.Host+"/")
			}
			if clone.Header.Get("Origin") == "" {
				clone.Header.Set("Origin", clone.URL.Scheme+"://"+clone.URL.Host)
			}
		}
	}

	return t.base.RoundTrip(clone)
}

// WebDAVClient provides WebDAV operations with dual-client strategy for performance optimization:
// - metadataClient: for metadata operations (list, query) with Keep-Alive enabled for better performance
// - streamClient: for file transfers with Keep-Alive disabled to avoid connection issues with large files
type WebDAVClient struct {
	metadataClient *gowebdav.Client // Metadata operations client (Keep-Alive enabled)
	streamClient   *gowebdav.Client // File streaming client (Keep-Alive disabled)
	url            string
	httpClient     *http.Client
	username       string // Store credentials for HTTP client
	password       string
}

// NewWebDAVClient creates a new WebDAV client with dual-client strategy
// to balance performance and stability:
// 1. Metadata operations (list, query) use Keep-Alive enabled client to reduce connection overhead
// 2. File transfer operations use Keep-Alive disabled client to avoid idle channel panics with large files
func NewWebDAVClient(serverURL, user, password string) *WebDAVClient {
	// Create transport for metadata operations (Keep-Alive enabled)
	metadataTransport := createTransport(true)
	metadataClient := gowebdav.NewClient(serverURL, user, password)
	metadataClient.SetTransport(&customTransport{base: metadataTransport})
	metadataClient.SetTimeout(DefaultMetadataTimeout)
	metadataClient.SetHeader("User-Agent", BrowserUserAgent)

	// Create transport for file transfers (Keep-Alive disabled)
	streamTransport := createTransport(false)
	streamClient := gowebdav.NewClient(serverURL, user, password)
	streamClient.SetTransport(&customTransport{base: streamTransport})
	streamClient.SetTimeout(0) // No timeout for file transfers
	streamClient.SetHeader("User-Agent", BrowserUserAgent)

	// HTTP client for advanced operations
	httpClient := &http.Client{
		Transport: &customTransport{base: streamTransport},
		Timeout:   0,
	}

	return &WebDAVClient{
		metadataClient: metadataClient,
		streamClient:   streamClient,
		url:            serverURL,
		httpClient:     httpClient,
		username:       user,
		password:       password,
	}
}

// createTransport creates a configured HTTP transport
// enableKeepAlive: true enables Keep-Alive (for metadata operations), false disables it (for file transfers)
func createTransport(enableKeepAlive bool) *http.Transport {
	var baseTransport *http.Transport
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		baseTransport = defaultTransport.Clone()
	} else {
		baseTransport = &http.Transport{}
	}

	baseTransport.MaxIdleConns = MaxIdleConns
	baseTransport.IdleConnTimeout = IdleConnTimeout
	baseTransport.DisableCompression = false
	baseTransport.DisableKeepAlives = !enableKeepAlive // Enable/disable Keep-Alive based on parameter
	baseTransport.MaxIdleConnsPerHost = MaxIdleConnsPerHost
	// Add strict concurrency limits per host to prevent overwhelming WebDAV servers
	baseTransport.MaxConnsPerHost = MaxConnsPerHost
	// Disable HTTP/2 for better compatibility
	baseTransport.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)

	return baseTransport
}

// TestConnection tests if the WebDAV connection is available
// Uses metadata client for connection testing as it's a lightweight operation
func (c *WebDAVClient) TestConnection() error {
	return c.metadataClient.Connect()
}

// MkdirAll creates a directory and all necessary parent directories
// Similar to mkdir -p, it creates intermediate directories as needed
func (c *WebDAVClient) MkdirAll(remotePath string) error {
	// Normalize path
	remotePath = strings.Trim(remotePath, "/")
	if remotePath == "" {
		return nil // Root directory always exists
	}

	targetPath := "/" + remotePath

	// OPTIMIZATION: Try fast-path first. Attempt to create the final directory directly.
	err := c.metadataClient.Mkdir(targetPath, 0755)
	if err == nil {
		return nil // Successfully created directly (parent existed)
	}

	// Check if it failed because it already exists
	if _, statErr := c.metadataClient.Stat(targetPath); statErr == nil {
		return nil // Already exists
	}

	// Fast path failed, fallback to step-by-step creation
	parts := strings.Split(remotePath, "/")
	current := ""

	// Create each directory level
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = path.Join(current, part)
		fullPath := "/" + current

		// Try to create directory directly without prior stat to save one PROPFIND if it doesn't exist
		err = c.metadataClient.Mkdir(fullPath, 0755)
		if err != nil {
			// Check if error is "already exists"
			if _, statErr := c.metadataClient.Stat(fullPath); statErr == nil {
				// Directory exists now, continue
				continue
			}
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	return nil
}

// sanitizePath properly escapes special characters in path while preserving directory structure
// Note: gowebdav library handles URL encoding internally, so we only need to handle edge cases
func sanitizePath(remotePath string) string {
	// gowebdav handles encoding internally, return path as-is
	// Only ensure the path starts with /
	if !strings.HasPrefix(remotePath, "/") {
		return "/" + remotePath
	}
	return remotePath
}

// ScanRemoteFiles recursively scans the remote directory for supported files
// Uses metadata client for directory listing operations, leveraging Keep-Alive for better performance
func (c *WebDAVClient) ScanRemoteFiles(dir string) ([]store.RemoteFile, error) {
	return c.scanRecursive(dir, 0)
}

func (c *WebDAVClient) scanRecursive(dir string, depth int) ([]store.RemoteFile, error) {
	if depth > MaxDirectoryScanDepth { // Depth limit to prevent infinite loops or excessively deep scans
		return nil, nil
	}

	var files []store.RemoteFile

	if dir == "" {
		dir = "/"
	}

	// Use metadata client for directory reading (Keep-Alive enabled)
	infos, err := c.metadataClient.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, info := range infos {
		fullPath := path.Join(dir, info.Name())

		if info.IsDir() {
			// CRITICAL: Skip metadata directory to prevent scanning bucket files as user content
			if info.Name() == MetadataDirectoryName {
				continue
			}
			// Recursive scan
			subFiles, err := c.scanRecursive(fullPath, depth+1)
			if err == nil {
				files = append(files, subFiles...)
			}
		} else {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext == ".pdf" || ext == ".gp" || ext == ".gp3" || ext == ".gp4" || ext == ".gp5" || ext == ".gpx" || ext == ".xml" || ext == ".musicxml" || ext == ".mxl" {
				files = append(files, store.RemoteFile{
					Name:  info.Name(),
					Path:  fullPath,
					Size:  info.Size(),
					IsDir: false,
				})
			}
		}
	}

	return files, nil
}

// ListRemoteDirectories returns a list of directories in the given path (non-recursive)
// Uses metadata client for directory listing operations
func (c *WebDAVClient) ListRemoteDirectories(dir string) ([]string, error) {
	var dirs []string

	if dir == "" {
		dir = "/"
	}

	// Use metadata client (Keep-Alive enabled)
	infos, err := c.metadataClient.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, info := range infos {
		if info.IsDir() {
			// CRITICAL: Filter out metadata directory to prevent recursive volume creation
			if info.Name() == MetadataDirectoryName {
				continue
			}
			fullPath := path.Join(dir, info.Name())
			dirs = append(dirs, fullPath)
		}
	}

	return dirs, nil
}

// ListDir returns a list of files and directories in the given path (non-recursive)
// Uses metadata client for directory listing operations
func (c *WebDAVClient) ListDir(dir string) ([]store.RemoteFile, error) {
	if dir == "" {
		dir = "/"
	}

	// Use metadata client (Keep-Alive enabled)
	infos, err := c.metadataClient.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var files []store.RemoteFile
	for _, info := range infos {
		fullPath := path.Join(dir, info.Name())

		isDir := info.IsDir()
		ext := strings.ToLower(filepath.Ext(info.Name()))

		// Include directories or supported files
		if isDir || ext == ".pdf" || ext == ".gp" || ext == ".gp3" || ext == ".gp4" || ext == ".gp5" || ext == ".gpx" || ext == ".xml" || ext == ".musicxml" || ext == ".mxl" {
			files = append(files, store.RemoteFile{
				Name:  info.Name(),
				Path:  fullPath,
				Size:  info.Size(),
				IsDir: isDir,
			})
		}
	}

	return files, nil
}

// DownloadFile downloads a single file to the local destination
// Uses stream client (Keep-Alive disabled) to avoid connection issues with large file transfers
func (c *WebDAVClient) DownloadFile(remotePath, localPath string) error {
	// Sanitize the remote path (gowebdav handles encoding internally)
	sanitizedPath := sanitizePath(remotePath)
	fmt.Printf("[WebDAV] DownloadFile: path=%s\n", sanitizedPath)

	// Use stream client for file download (Keep-Alive disabled)
	data, err := c.streamClient.ReadStream(sanitizedPath)
	if err != nil {
		fmt.Printf("[WebDAV] DownloadFile failed for %s: %v\n", sanitizedPath, err)
		return fmt.Errorf("failed to download %s: %w", remotePath, err)
	}
	defer data.Close()

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localPath, err)
	}
	defer out.Close()

	written, err := io.Copy(out, data)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", localPath, err)
	}
	fmt.Printf("[WebDAV] DownloadFile completed: %d bytes written to %s\n", written, localPath)
	return nil
}

// UploadFile uploads a single file to the remote directory
// Uses stream client (Keep-Alive disabled) to avoid connection issues with large file transfers
func (c *WebDAVClient) UploadFile(localPath, remoteDir string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer f.Close()

	// Ensure remote directory exists (simple check, might fail if parent doesn't exist)
	// gowebdav doesn't provide MkdirAll, only Mkdir. Recursive mkdir is complex.
	// We assume the user picked an existing dir from ListRemoteDirectories or root
	c.metadataClient.Mkdir(remoteDir, 0755)

	fileName := filepath.Base(localPath)
	remotePath := path.Join(remoteDir, fileName)

	// Use stream client for file upload (Keep-Alive disabled)
	return c.streamClient.WriteStream(remotePath, f, 0644)
}

// ReadStream returns a read stream for the remote file (for streaming/proxy)
// Uses stream client (Keep-Alive disabled) to avoid connection issues with large file transfers
func (c *WebDAVClient) ReadStream(remotePath string) (io.ReadCloser, error) {
	sanitizedPath := sanitizePath(remotePath)
	fmt.Printf("[WebDAV] ReadStream: path=%s\n", sanitizedPath)
	// Use stream client for file stream reading (Keep-Alive disabled)
	return c.streamClient.ReadStream(sanitizedPath)
}

// GetFileInfo returns file info for a remote path
// Uses metadata client for file info queries
func (c *WebDAVClient) GetFileInfo(remotePath string) (os.FileInfo, error) {
	sanitizedPath := sanitizePath(remotePath)
	// Use metadata client (Keep-Alive enabled)
	return c.metadataClient.Stat(sanitizedPath)
}

// GetHTTPClient returns the underlying HTTP client for advanced operations
func (c *WebDAVClient) GetHTTPClient() *http.Client {
	return c.httpClient
}

// GetURL returns the base WebDAV URL
func (c *WebDAVClient) GetURL() string {
	return c.url
}

// ReadBytes reads all bytes from a remote file path.
func (c *WebDAVClient) ReadBytes(remotePath string) ([]byte, error) {
	stream, err := c.ReadStream(remotePath)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return io.ReadAll(stream)
}

// WriteBytes writes bytes to a remote file path, creating parent directories if needed.
func (c *WebDAVClient) WriteBytes(remotePath string, data []byte) error {
	remotePath = sanitizePath(remotePath)
	parentDir := path.Dir(remotePath)
	if parentDir != "." && parentDir != "/" {
		if err := c.MkdirAll(parentDir); err != nil {
			return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
		}
	}

	reader := bytes.NewReader(data)
	if err := c.streamClient.WriteStream(remotePath, reader, 0644); err != nil {
		return fmt.Errorf("failed to write remote file %s: %w", remotePath, err)
	}
	return nil
}
