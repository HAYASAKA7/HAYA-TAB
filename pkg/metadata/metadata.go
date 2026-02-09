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

// DownloadCover searches iTunes and saves the cover to dstPath.
// Falls back to US/en_us if specific country/lang returns no results.
func DownloadCover(artist, album, title, country, lang, dstPath string) error {
	// 1. Try with user params
	if country == "" {
		country = "US"
	}
	if lang == "" {
		lang = "en_us"
	}

	err := attemptDownload(artist, album, title, country, lang, dstPath)
	if err == nil {
		return nil
	}

	// 2. Fallback to US if different
	if country != "US" {
		fmt.Printf("Search failed for %s/%s, falling back to US...\n", country, lang)
		return attemptDownload(artist, album, title, "US", "en_us", dstPath)
	}

	return err
}

func attemptDownload(artist, album, title, country, lang, dstPath string) error {
	var term, entity string
	if album != "" {
		term = artist + " " + album
		entity = "album"
	} else {
		term = artist + " " + title
		entity = "song"
	}

	query := url.QueryEscape(term)
	apiURL := fmt.Sprintf("https://itunes.apple.com/search?term=%s&entity=%s&limit=1&country=%s&lang=%s", query, entity, country, lang)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("iTunes API error: status code %d", resp.StatusCode)
	}

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
	imgReq, err := http.NewRequest("GET", artworkURL, nil)
	if err != nil {
		return err
	}
	imgReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	imgResp, err := client.Do(imgReq)
	if err != nil {
		return err
	}
	defer imgResp.Body.Close()

	// Ensure directory exists
	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create covers directory: %w", err)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, imgResp.Body)
	return err
}
