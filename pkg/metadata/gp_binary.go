package metadata

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// parseGPBinary attempts to parse GP3, GP4, GP5 files
func parseGPBinary(path string) (Metadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return Metadata{}, err
	}
	defer f.Close()

	// 1. Read Version (30 bytes)
	versionBuf := make([]byte, 30)
	if _, err := io.ReadFull(f, versionBuf); err != nil {
		return Metadata{}, err
	}
	
	// Truncate null bytes and clean
	versionRaw := string(versionBuf)
	if idx := indexOf(versionBuf, 0); idx != -1 {
		versionRaw = versionRaw[:idx]
	}
	version := strings.TrimSpace(versionRaw)

	if !validVersion(version) {
		return Metadata{}, fmt.Errorf("unknown GP version: %s", version)
	}

	// Determine version for string reading strategy
	// Format: "FICHIER GUITAR PRO vX.YZ"
	var majorVersion int
	// Find "v"
	vIdx := strings.LastIndex(version, "v")
	if vIdx != -1 && vIdx+1 < len(version) {
		fmt.Sscanf(version[vIdx+1:], "%d", &majorVersion)
	}

	// Strategy:
	// GP3: 1 byte length
	// GP4/5: 4 byte length (int32)

	readString := func() (string, error) {
		var length int
		
		if majorVersion < 4 {
			// GP3: 1 byte length
			var l uint8
			if err := binary.Read(f, binary.LittleEndian, &l); err != nil {
				return "", err
			}
			length = int(l)
		} else {
			// GP4/5: 4 byte length
			var l int32
			if err := binary.Read(f, binary.LittleEndian, &l); err != nil {
				return "", err
			}
			length = int(l)
		}

		if length == 0 {
			return "", nil
		}
		
		// Sanity check
		if length < 0 || length > 2048 {
			return "", fmt.Errorf("invalid string length: %d", length)
		}

		buf := make([]byte, length)
		if _, err := io.ReadFull(f, buf); err != nil {
			return "", err
		}
		// NOTE: Real implementation should handle Charset (CP1252), but for now raw string
		return string(buf), nil
	}
	
	// GP5 often has score info immediately after version?
	// The structure for GP3/4/5 generally starts with:
	// - Version (30 bytes)
	// - Score Information Block
	//   - Title
	//   - Subtitle
	//   - Artist
	//   - Album
	//   - Author
	//   ...
	
	// However, some versions might have extra bytes.
	// For robustness, if the first read fails or looks like garbage, we might need a better parser.
	// But assuming standard file integrity:

	var m Metadata

	// Title
	title, err := readString()
	if err != nil { return Metadata{}, fmt.Errorf("failed to read title: %w", err) }
	m.Title = title

	// Subtitle (skip)
	_, err = readString()
	if err != nil { return Metadata{}, fmt.Errorf("failed to read subtitle: %w", err) }

	// Artist
	artist, err := readString()
	if err != nil { return Metadata{}, fmt.Errorf("failed to read artist: %w", err) }
	m.Artist = artist

	// Album
	album, err := readString()
	if err != nil { return Metadata{}, fmt.Errorf("failed to read album: %w", err) }
	m.Album = album
	
	return m, nil
}

func indexOf(data []byte, b byte) int {
	for i, v := range data {
		if v == b {
			return i
		}
	}
	return -1
}

func validVersion(v string) bool {
	// e.g. "FICHIER GUITAR PRO v5.00"
	return strings.HasPrefix(v, "FICHIER GUITAR PRO")
}
