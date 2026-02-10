package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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

// Regex patterns for common filename formats
var (
	// "01. Artist - Title.ext" or "01 Artist - Title.ext"
	trackNumberPattern = regexp.MustCompile(`^(\d{1,3})[.\s]+(.+)$`)

	// "[Artist] Title.ext"
	bracketArtistPattern = regexp.MustCompile(`^\[([^\]]+)\]\s*(.+)$`)

	// "Artist - Title (Key).ext" or "Artist - Title [Key].ext"
	keyPattern = regexp.MustCompile(`^(.+?)\s*[\(\[]([A-Ga-g][#b]?(?:\s*(?:major|minor|m|M))?|[A-Ga-g][#b]?m?)[\)\]]$`)

	// "Title - Artist.ext" (reversed format, common in some regions)
	// We'll detect this by checking if the second part looks more like an artist name
)

// ParseFilename attempts to extract Artist - Album - Song from filename
// Enhanced with multiple pattern recognition:
// 1. "Artist - Album - Title.ext"
// 2. "Artist - Title.ext"
// 3. "01. Artist - Title.ext" (with track number)
// 4. "[Artist] Title.ext" (bracket format)
// 5. "Artist - Title (Key).ext" (with key signature)
// 6. "Title.ext" (fallback)
func ParseFilename(filename string) Metadata {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Clean up common artifacts
	name = cleanFilename(name)

	m := Metadata{Title: name} // Default

	// Try bracket format first: "[Artist] Title"
	if matches := bracketArtistPattern.FindStringSubmatch(name); len(matches) == 3 {
		m.Artist = strings.TrimSpace(matches[1])
		m.Title = strings.TrimSpace(matches[2])
		// Remove key from title if present
		m.Title = removeKeyFromTitle(m.Title)
		return m
	}

	// Remove track number prefix if present
	workingName := name
	if matches := trackNumberPattern.FindStringSubmatch(name); len(matches) == 3 {
		workingName = strings.TrimSpace(matches[2])
	}

	// Split by " - " (with spaces around dash)
	parts := splitByDash(workingName)

	if len(parts) >= 3 {
		// "Artist - Album - Title" format
		m.Artist = parts[0]
		m.Album = parts[1]
		m.Title = removeKeyFromTitle(parts[2])
	} else if len(parts) == 2 {
		// "Artist - Title" format
		m.Artist = parts[0]
		m.Title = removeKeyFromTitle(parts[1])
	} else {
		// Single part - just title
		m.Title = removeKeyFromTitle(workingName)
	}

	return m
}

// splitByDash splits a string by dash separators, handling various dash formats
func splitByDash(s string) []string {
	// Try " - " first (most common)
	if strings.Contains(s, " - ") {
		parts := strings.Split(s, " - ")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	// Try " – " (en-dash)
	if strings.Contains(s, " – ") {
		parts := strings.Split(s, " – ")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	// Try " — " (em-dash)
	if strings.Contains(s, " — ") {
		parts := strings.Split(s, " — ")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	// Fallback to simple dash (but only if it looks like a separator)
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	return []string{s}
}

// removeKeyFromTitle removes key signatures from the end of a title
func removeKeyFromTitle(title string) string {
	if matches := keyPattern.FindStringSubmatch(title); len(matches) == 3 {
		return strings.TrimSpace(matches[1])
	}
	return title
}

// cleanFilename removes common artifacts from filenames
func cleanFilename(name string) string {
	// Remove common suffixes
	suffixes := []string{
		" (Official)", " (Official Audio)", " (Official Video)",
		" (Lyrics)", " (Lyric Video)", " (Audio)",
		" (HD)", " (HQ)", " (4K)",
		" [Official]", " [Official Audio]", " [Official Video]",
		" [Lyrics]", " [Lyric Video]", " [Audio]",
		" [HD]", " [HQ]", " [4K]",
	}

	result := name
	for _, suffix := range suffixes {
		if strings.HasSuffix(strings.ToLower(result), strings.ToLower(suffix)) {
			result = result[:len(result)-len(suffix)]
		}
	}

	// Remove leading/trailing whitespace
	result = strings.TrimSpace(result)

	return result
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
