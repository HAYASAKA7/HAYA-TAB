package metadata

import (
	"testing"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     Metadata
	}{
		{
			name:     "Artist - Title format",
			filename: "Led Zeppelin - Stairway to Heaven.pdf",
			want: Metadata{
				Title:  "Stairway to Heaven",
				Artist: "Led Zeppelin",
				Album:  "",
			},
		},
		{
			name:     "Artist - Album - Title format",
			filename: "The Beatles - Abbey Road - Come Together.gp5",
			want: Metadata{
				Title:  "Come Together",
				Artist: "The Beatles",
				Album:  "Abbey Road",
			},
		},
		{
			name:     "Track number prefix",
			filename: "01. Pink Floyd - Comfortably Numb.pdf",
			want: Metadata{
				Title:  "Comfortably Numb",
				Artist: "Pink Floyd",
				Album:  "",
			},
		},
		{
			name:     "Bracket artist format",
			filename: "[Queen] Bohemian Rhapsody.gp",
			want: Metadata{
				Title:  "Bohemian Rhapsody",
				Artist: "Queen",
				Album:  "",
			},
		},
		{
			name:     "Title with key signature",
			filename: "Metallica - Enter Sandman (Em).gpx",
			want: Metadata{
				Title:  "Enter Sandman",
				Artist: "Metallica",
				Album:  "",
			},
		},
		{
			name:     "Title only",
			filename: "Amazing Song.pdf",
			want: Metadata{
				Title:  "Amazing Song",
				Artist: "",
				Album:  "",
			},
		},
		{
			name:     "With path",
			filename: "/path/to/AC DC - Back in Black.gp5",
			want: Metadata{
				Title:  "Back in Black",
				Artist: "AC DC",
				Album:  "",
			},
		},
		{
			name:     "Clean artifacts",
			filename: "Nirvana - Smells Like Teen Spirit (Official Audio).pdf",
			want: Metadata{
				Title:  "Smells Like Teen Spirit",
				Artist: "Nirvana",
				Album:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFilename(tt.filename)
			if got.Title != tt.want.Title {
				t.Errorf("ParseFilename() Title = %v, want %v", got.Title, tt.want.Title)
			}
			if got.Artist != tt.want.Artist {
				t.Errorf("ParseFilename() Artist = %v, want %v", got.Artist, tt.want.Artist)
			}
			if got.Album != tt.want.Album {
				t.Errorf("ParseFilename() Album = %v, want %v", got.Album, tt.want.Album)
			}
		})
	}
}

func TestCleanFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Remove Official Audio",
			input: "Song Title (Official Audio)",
			want:  "Song Title",
		},
		{
			name:  "Remove Lyrics",
			input: "Song Title [Lyrics]",
			want:  "Song Title",
		},
		{
			name:  "Remove HD",
			input: "Song Title (HD)",
			want:  "Song Title",
		},
		{
			name:  "No artifacts",
			input: "Clean Title",
			want:  "Clean Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanFilename(tt.input)
			if got != tt.want {
				t.Errorf("cleanFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitByDash(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "Standard dash with spaces",
			input: "Artist - Title",
			want:  []string{"Artist", "Title"},
		},
		{
			name:  "Three parts",
			input: "Artist - Album - Title",
			want:  []string{"Artist", "Album", "Title"},
		},
		{
			name:  "En-dash",
			input: "Artist – Title",
			want:  []string{"Artist", "Title"},
		},
		{
			name:  "Em-dash",
			input: "Artist — Title",
			want:  []string{"Artist", "Title"},
		},
		{
			name:  "No dash",
			input: "Single Title",
			want:  []string{"Single Title"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitByDash(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitByDash() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitByDash()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRemoveKeyFromTitle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Key in parentheses",
			input: "Song Title (Em)",
			want:  "Song Title",
		},
		{
			name:  "Key in brackets",
			input: "Song Title [A major]",
			want:  "Song Title",
		},
		{
			name:  "Sharp key",
			input: "Song Title (C#m)",
			want:  "Song Title",
		},
		{
			name:  "Flat key",
			input: "Song Title (Bb)",
			want:  "Song Title",
		},
		{
			name:  "No key",
			input: "Song Title",
			want:  "Song Title",
		},
		{
			name:  "Not a key",
			input: "Song Title (Live)",
			want:  "Song Title (Live)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeKeyFromTitle(tt.input)
			if got != tt.want {
				t.Errorf("removeKeyFromTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid PDF path",
			path:    "test.pdf",
			wantErr: false,
		},
		{
			name:    "Valid GP path",
			path:    "test.gp5",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Title == "" {
				t.Errorf("ParseFile() returned empty title")
			}
		})
	}
}
