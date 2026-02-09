package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Metadata struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
}

type ItunesResponse struct {
	ResultCount int `json:"resultCount"`
	Results     []struct {
		ArtworkUrl100 string `json:"artworkUrl100"`
	} `json:"results"`
}

// ParseFilename attempts to extract Artist - Album - Song from filename
// Heuristics:
// 1. "Artist - Album - Title.ext"
// 2. "Artist - Title.ext"
// 3. "Title.ext"
func ParseFilename(filename string) Metadata {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	parts := strings.Split(name, "-")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	m := Metadata{Title: name} // Default

	if len(parts) >= 3 {
		m.Artist = parts[0]
		m.Album = parts[1]
		m.Title = parts[2]
	} else if len(parts) == 2 {
		m.Artist = parts[0]
		m.Title = parts[1]
	}

	return m
}

// DownloadCover searches iTunes and saves the cover to dstPath
func DownloadCover(artist, album string, dstPath string) error {
	term := artist + " " + album
	query := url.QueryEscape(term)
	apiURL := fmt.Sprintf("https://itunes.apple.com/search?term=%s&entity=album&limit=1", query)

	resp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result ItunesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.ResultCount == 0 || len(result.Results) == 0 {
		return fmt.Errorf("no results found")
	}

	artworkURL := result.Results[0].ArtworkUrl100
	// Try to get higher res
	artworkURL = strings.Replace(artworkURL, "100x100bb", "600x600bb", 1)

	// Download
	imgResp, err := http.Get(artworkURL)
	if err != nil {
		return err
	}
	defer imgResp.Body.Close()

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, imgResp.Body)
	return err
}
