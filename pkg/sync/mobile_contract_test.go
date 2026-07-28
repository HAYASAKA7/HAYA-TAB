package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMobileFingerprintFixturesMatchDesktopContract(t *testing.T) {
	root := filepath.Join("..", "..", "shared", "fixtures", "fingerprint", "v1")

	data0, err := os.ReadFile(filepath.Join(root, "bucket-00.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bucket0 struct {
		Metadata FingerprintMetadata `json:"metadata"`
		Files    []FingerprintFile   `json:"files"`
	}
	if err := json.Unmarshal(data0, &bucket0); err != nil {
		t.Fatal(err)
	}
	if bucket0.Metadata.VolumeID == "" || bucket0.Metadata.BucketCount != BucketCount {
		t.Fatalf("invalid bucket-00 metadata: %#v", bucket0.Metadata)
	}

	data3, err := os.ReadFile(filepath.Join(root, "bucket-03.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bucket3 BucketData
	if err := json.Unmarshal(data3, &bucket3); err != nil {
		t.Fatal(err)
	}
	if bucket3.BucketNumber != 3 {
		t.Fatalf("bucket_number = %d, want 3", bucket3.BucketNumber)
	}
	for _, file := range append(bucket0.Files, bucket3.Files...) {
		if file.RelativePath == "" || file.Title == "" || file.Type == "" {
			t.Fatalf("incomplete file %#v", file)
		}
	}
}

func TestMobileFingerprintInvalidFixtureIsRejected(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(
		"..", "..", "shared", "fixtures", "fingerprint", "v1", "bucket.invalid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var bucket BucketData
	if err := json.Unmarshal(data, &bucket); err == nil {
		t.Fatal("invalid fixture unexpectedly decoded")
	}
}
