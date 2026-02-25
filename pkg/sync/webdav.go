package sync

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"haya-tab/pkg/store"

	"github.com/studio-b12/gowebdav"
)

type WebDAVClient struct {
	client *gowebdav.Client
	url    string
}

func NewWebDAVClient(url, user, password string) *WebDAVClient {
	return &WebDAVClient{
		client: gowebdav.NewClient(url, user, password),
		url:    url,
	}
}

func (c *WebDAVClient) TestConnection() error {
	return c.client.Connect()
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
	data, err := c.client.ReadStream(remotePath)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", remotePath, err)
	}
	defer data.Close()

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, data)
	return err
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
