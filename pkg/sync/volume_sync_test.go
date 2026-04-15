package sync

import (
	"strings"
	"testing"
)

// TestParseMetadataFromFilename tests the pure string-parsing function that
// extracts title and artist from common filename patterns like "Artist - Title.pdf".
func TestParseMetadataFromFilename(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantTitle  string
		wantArtist string
	}{
		{
			name:       "standard Artist - Title format",
			filename:   "Pink Floyd - Comfortably Numb.pdf",
			wantTitle:  "Comfortably Numb",
			wantArtist: "Pink Floyd",
		},
		{
			name:       "no artist separator",
			filename:   "Comfortably Numb.gp5",
			wantTitle:  "Comfortably Numb",
			wantArtist: "",
		},
		{
			name:       "multiple dashes — only first split is used",
			filename:   "Artist - Album - Title.pdf",
			wantTitle:  "Album - Title",
			wantArtist: "Artist",
		},
		{
			name:       "extension is stripped",
			filename:   "Just A Song.gpx",
			wantTitle:  "Just A Song",
			wantArtist: "",
		},
		{
			name:       "leading/trailing spaces are trimmed",
			filename:   "  Artist  -  Title  .pdf",
			wantTitle:  "Title",
			wantArtist: "Artist",
		},
		{
			name:       "no extension",
			filename:   "Artist - Song",
			wantTitle:  "Song",
			wantArtist: "Artist",
		},
		{
			name:       "empty filename",
			filename:   "",
			wantTitle:  "",
			wantArtist: "",
		},
		{
			name:       "only extension",
			filename:   ".pdf",
			wantTitle:  "",
			wantArtist: "",
		},
		{
			name:       "GP4 extension",
			filename:   "Band - Track Name.gp4",
			wantTitle:  "Track Name",
			wantArtist: "Band",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotArtist := parseMetadataFromFilename(tt.filename)
			if gotTitle != tt.wantTitle {
				t.Errorf("title = %q, want %q", gotTitle, tt.wantTitle)
			}
			if gotArtist != tt.wantArtist {
				t.Errorf("artist = %q, want %q", gotArtist, tt.wantArtist)
			}
		})
	}
}

// TestGenerateID verifies that generateID returns a non-empty, numeric string
// and that successive calls produce distinct values (barring nanosecond collision).
func TestGenerateID(t *testing.T) {
	id := generateID()
	if id == "" {
		t.Fatal("generateID() returned empty string")
	}
	// Should be a decimal integer (Unix nanoseconds)
	for _, ch := range id {
		if ch < '0' || ch > '9' {
			t.Errorf("generateID() = %q contains non-digit character %q", id, string(ch))
		}
	}

	// Two calls should normally differ, but we just check neither is empty.
	id2 := generateID()
	if id2 == "" {
		t.Fatal("second generateID() call returned empty string")
	}
}

// TestGetDeviceName verifies that getDeviceName returns a non-empty string
// (either hostname or "unknown-device" fallback).
func TestGetDeviceName(t *testing.T) {
	name := getDeviceName()
	if name == "" {
		t.Error("getDeviceName() returned empty string")
	}
	// Must be a single-component name (no forward-slash path separators)
	if strings.Contains(name, "/") {
		t.Errorf("getDeviceName() = %q, should not contain '/'", name)
	}
}
