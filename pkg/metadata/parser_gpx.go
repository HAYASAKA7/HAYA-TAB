package metadata

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type GpifScore struct {
	Title  string `xml:"Score>Title"`
	Artist string `xml:"Score>Artist"`
	Album  string `xml:"Score>Album"`
}

type GpifRoot struct {
	Score GpifScore `xml:"Score"`
}

// parseGPX parses .gpx files (zipped XML)
func parseGPX(path string) (Metadata, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return Metadata{}, err
	}
	defer r.Close()

	var scoreFile *zip.File
	for _, f := range r.File {
		if strings.EqualFold(f.Name, "score.gpif") || strings.HasSuffix(strings.ToLower(f.Name), "/score.gpif") {
			scoreFile = f
			break
		}
	}

	if scoreFile == nil {
		return Metadata{}, fmt.Errorf("score.gpif not found in gpx file")
	}

	rc, err := scoreFile.Open()
	if err != nil {
		return Metadata{}, err
	}
	defer rc.Close()

	// Read content
	// Limit to reasonable size to prevent bombs
	content, err := io.ReadAll(io.LimitReader(rc, 10*1024*1024)) // 10MB limit
	if err != nil {
		return Metadata{}, err
	}

	var root GpifRoot
	if err := xml.Unmarshal(content, &root); err != nil {
		return Metadata{}, err
	}

	return Metadata{
		Title:  root.Score.Title,
		Artist: root.Score.Artist,
		Album:  root.Score.Album,
	}, nil
}
