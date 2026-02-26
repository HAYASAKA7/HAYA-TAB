package sync

import (
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

type WebDAVClient struct {
	client     *gowebdav.Client
	url        string
	httpClient *http.Client
}

func NewWebDAVClient(serverURL, user, password string) *WebDAVClient {
	// Create custom HTTP client with browser-like transport and long timeout for large files
	var baseTransport *http.Transport
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		baseTransport = defaultTransport.Clone()
	} else {
		baseTransport = &http.Transport{}
	}

	baseTransport.MaxIdleConns = 10
	baseTransport.IdleConnTimeout = 90 * time.Second
	baseTransport.DisableCompression = false
	baseTransport.DisableKeepAlives = true // Disable Keep-Alives to prevent idle HTTP channel panics during file streaming
	baseTransport.MaxIdleConnsPerHost = 5
	// Disable HTTP/2
	baseTransport.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)

	transport := &customTransport{
		base: baseTransport,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   0, // Disable timeout for large file transfers
	}

	client := gowebdav.NewClient(serverURL, user, password)
	// Set the custom HTTP client with browser User-Agent
	client.SetTransport(transport)
	client.SetTimeout(0)
	// Also set header directly as fallback
	client.SetHeader("User-Agent", BrowserUserAgent)

	return &WebDAVClient{
		client:     client,
		url:        serverURL,
		httpClient: httpClient,
	}
}

func (c *WebDAVClient) TestConnection() error {
	return c.client.Connect()
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

// encodePathForURL encodes path segments for direct HTTP requests (not gowebdav)
func encodePathForURL(remotePath string) string {
	// Split path into segments
	segments := strings.Split(remotePath, "/")

	// Escape each segment individually
	for i, segment := range segments {
		if segment != "" {
			segments[i] = url.PathEscape(segment)
		}
	}

	// Rejoin with slashes
	encoded := strings.Join(segments, "/")

	// Ensure leading slash is preserved
	if strings.HasPrefix(remotePath, "/") && !strings.HasPrefix(encoded, "/") {
		encoded = "/" + encoded
	}

	return encoded
}

// ScanRemoteFiles recursively scans the remote directory for supported files
func (c *WebDAVClient) ScanRemoteFiles(dir string) ([]store.RemoteFile, error) {
	return c.scanRecursive(dir, 0)
}

func (c *WebDAVClient) scanRecursive(dir string, depth int) ([]store.RemoteFile, error) {
	if depth > 10 { // Depth limit to prevent infinite loops or excessively deep scans
		return nil, nil
	}

	var files []store.RemoteFile

	if dir == "" {
		dir = "/"
	}

	infos, err := c.client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, info := range infos {
		fullPath := path.Join(dir, info.Name())

		if info.IsDir() {
			// Recursive scan
			subFiles, err := c.scanRecursive(fullPath, depth+1)
			if err == nil {
				files = append(files, subFiles...)
			}
		} else {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext == ".gp" || ext == ".gp5" || ext == ".gpx" || ext == ".pdf" {
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
func (c *WebDAVClient) ListRemoteDirectories(dir string) ([]string, error) {
	var dirs []string

	if dir == "" {
		dir = "/"
	}

	infos, err := c.client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, info := range infos {
		if info.IsDir() {
			fullPath := path.Join(dir, info.Name())
			dirs = append(dirs, fullPath)
		}
	}

	return dirs, nil
}

// ListDir returns a list of files and directories in the given path (non-recursive)
func (c *WebDAVClient) ListDir(dir string) ([]store.RemoteFile, error) {
	if dir == "" {
		dir = "/"
	}

	infos, err := c.client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var files []store.RemoteFile
	for _, info := range infos {
		fullPath := path.Join(dir, info.Name())

		isDir := info.IsDir()
		ext := strings.ToLower(filepath.Ext(info.Name()))

		// Include directories or supported files
		if isDir || ext == ".gp" || ext == ".gp5" || ext == ".gpx" || ext == ".pdf" {
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
func (c *WebDAVClient) DownloadFile(remotePath, localPath string) error {
	// Sanitize the remote path (gowebdav handles encoding internally)
	sanitizedPath := sanitizePath(remotePath)
	fmt.Printf("[WebDAV] DownloadFile: path=%s\n", sanitizedPath)

	data, err := c.client.ReadStream(sanitizedPath)
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
func (c *WebDAVClient) UploadFile(localPath, remoteDir string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer f.Close()

	// Ensure remote directory exists (simple check, might fail if parent doesn't exist,
	// but gowebdav MkdirAll isn't available, only Mkdir. recursive mkdir is complex.
	// We'll assume the user picked an existing dir from ListRemoteDirectories or root)
	// For robustness, we could try to create it.
	c.client.Mkdir(remoteDir, 0755)

	fileName := filepath.Base(localPath)
	remotePath := path.Join(remoteDir, fileName)

	return c.client.WriteStream(remotePath, f, 0644)
}

// ReadStream returns a read stream for the remote file (for streaming/proxy)
func (c *WebDAVClient) ReadStream(remotePath string) (io.ReadCloser, error) {
	sanitizedPath := sanitizePath(remotePath)
	fmt.Printf("[WebDAV] ReadStream: path=%s\n", sanitizedPath)
	return c.client.ReadStream(sanitizedPath)
}

// GetFileInfo returns file info for a remote path
func (c *WebDAVClient) GetFileInfo(remotePath string) (os.FileInfo, error) {
	sanitizedPath := sanitizePath(remotePath)
	return c.client.Stat(sanitizedPath)
}

// EncodePathForURL encodes path for direct HTTP requests (exported for main.go)
func EncodePathForURL(remotePath string) string {
	return encodePathForURL(remotePath)
}

// GetHTTPClient returns the underlying HTTP client for advanced operations
func (c *WebDAVClient) GetHTTPClient() *http.Client {
	return c.httpClient
}

// GetURL returns the base WebDAV URL
func (c *WebDAVClient) GetURL() string {
	return c.url
}
