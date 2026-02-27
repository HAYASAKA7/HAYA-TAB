// Package metadata provides music metadata parsing and fetching utilities.
package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// MusicBrainz API base URL
	musicBrainzBaseURL = "https://musicbrainz.org/ws/2"
	// User-Agent required by MusicBrainz API
	musicBrainzUserAgent = "HAYA-TAB/2.0.0 ( contact@example.com )"
)

// MusicBrainzArtistResponse represents the response from MusicBrainz artist search
type MusicBrainzArtistResponse struct {
	Created string                  `json:"created"`
	Count   int                     `json:"count"`
	Offset  int                     `json:"offset"`
	Artists []MusicBrainzArtistInfo `json:"artists"`
}

// MusicBrainzArtistInfo represents an artist entry from MusicBrainz
type MusicBrainzArtistInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SortName    string `json:"sort-name"`
	Country     string `json:"country"`
	Type        string `json:"type"`
	Score       int    `json:"score"`
	Disambiguation string `json:"disambiguation"`
}

// MusicBrainzClient handles requests to the MusicBrainz API
type MusicBrainzClient struct {
	httpClient *http.Client
}

// NewMusicBrainzClient creates a new MusicBrainz API client
func NewMusicBrainzClient() *MusicBrainzClient {
	return &MusicBrainzClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchArtistCountry searches for an artist and returns their origin country code
// Returns empty string if not found or on error
func (c *MusicBrainzClient) SearchArtistCountry(artistName string) (string, error) {
	if artistName == "" {
		return "", fmt.Errorf("artist name is empty")
	}

	// Build the search query
	// Using simple query parameter for fuzzy matching (handles spacing variations)
	query := url.QueryEscape(artistName)
	apiURL := fmt.Sprintf("%s/artist/?query=%s&fmt=json&limit=5", musicBrainzBaseURL, query)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// MusicBrainz requires a meaningful User-Agent
	req.Header.Set("User-Agent", musicBrainzUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limited by MusicBrainz (status %d)", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("MusicBrainz API error: status %d", resp.StatusCode)
	}

	var result MusicBrainzArtistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Artists) == 0 {
		return "", fmt.Errorf("no artists found for: %s", artistName)
	}

	// Find the best match - prefer exact name match with country info
	for _, artist := range result.Artists {
		if artist.Country != "" {
			// Check for exact or close name match
			if strings.EqualFold(artist.Name, artistName) {
				return artist.Country, nil
			}
		}
	}

	// Fallback: return the first artist with a country
	for _, artist := range result.Artists {
		if artist.Country != "" {
			return artist.Country, nil
		}
	}

	return "", fmt.Errorf("no country info found for: %s", artistName)
}
