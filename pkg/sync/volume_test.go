package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebDAVClient_FingerprintOperations(t *testing.T) {
	// Mock server state
	files := make(map[string][]byte)
	dirs := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			dirs[r.URL.Path] = true
			w.WriteHeader(http.StatusCreated)
		case "PUT":
			// Simplified PUT - gowebdav might do more, but we just need it to succeed
			w.WriteHeader(http.StatusCreated)
		case "GET":
			if data, ok := files[r.URL.Path]; ok {
				w.Write(data)
			} else if r.URL.Path == "/vol1/haya-metadata/bucket-00.json" {
				// Return empty metadata for ReadMetadata
				type Bucket0 struct {
					Metadata FingerprintMetadata `json:"metadata"`
					Files    []FingerprintFile   `json:"files"`
				}
				b0 := Bucket0{
					Metadata: FingerprintMetadata{VolumeID: "vol1"},
					Files:    []FingerprintFile{},
				}
				data, _ := json.Marshal(b0)
				w.Write(data)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case "PROPFIND":
			// Always return that directory exists for simplicity
			w.WriteHeader(http.StatusMultiStatus)
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8" ?><D:multistatus xmlns:D="DAV:"><D:response><D:href>/</D:href><D:propstat><D:status>HTTP/1.1 200 OK</D:status></D:propstat></D:response></D:multistatus>`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewWebDAVClient(server.URL, "user", "pass")

	// Test CreateVolumeFingerprint (will call MkdirAll, WriteMetadata, WriteBucket)
	// Note: gowebdav's Mkdir/WriteStream might fail with this simple mock, 
	// but we're testing that the logic flow reaches these points.
	
	// We'll focus on testing the helper functions that don't depend on a perfect WebDAV server
	
	t.Run("HelperPaths", func(t *testing.T) {
		if getMetadataPath("/vol") != "/vol/haya-metadata" {
			t.Errorf("wrong metadata path: %s", getMetadataPath("/vol"))
		}
		if getBucketPath("/vol", 5) != "/vol/haya-metadata/bucket-05.json" {
			t.Errorf("wrong bucket path: %s", getBucketPath("/vol", 5))
		}
		if getLegacyFingerprintPath("/vol") != "/vol/.haya-volume-fingerprint" {
			t.Errorf("wrong legacy path: %s", getLegacyFingerprintPath("/vol"))
		}
	})

	t.Run("CalculateBucketNumber", func(t *testing.T) {
		b1 := CalculateBucketNumber("file1.pdf")
		b2 := CalculateBucketNumber("file2.pdf")
		if b1 < 0 || b1 >= 16 || b2 < 0 || b2 >= 16 {
			t.Errorf("bucket number out of range: %d, %d", b1, b2)
		}
	})
	
	// Since full WebDAV integration tests are hard with a simple mock, 
	// let's at least test the logic that can be hit.
	
	// Testing ReadMetadata with a mock response
	t.Run("ReadMetadata", func(t *testing.T) {
		metadata, err := client.ReadMetadata("/vol1")
		if err != nil {
			t.Fatalf("ReadMetadata failed: %v", err)
		}
		if metadata.VolumeID != "vol1" {
			t.Errorf("expected vol1, got %s", metadata.VolumeID)
		}
	})
}
