package metadata

import (
	"testing"
)

func TestCalculateInitials_Chinese(t *testing.T) {
	testCases := []struct {
		title          string
		originCountry  string
		expectedAZ     string
		expectedKana   string
	}{
		{"青春", "CN", "Q", "#"},
		{"爱情故事", "CN", "A", "#"},
		{"北京欢迎你", "TW", "B", "#"},
		{"香港之夜", "HK", "X", "#"},
		{"中国", "CN", "Z", "#"},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			az, kana := CalculateInitials(tc.title, tc.originCountry)
			if az != tc.expectedAZ {
				t.Errorf("CalculateInitials(%q, %q) az = %q, want %q", tc.title, tc.originCountry, az, tc.expectedAZ)
			}
			if kana != tc.expectedKana {
				t.Errorf("CalculateInitials(%q, %q) kana = %q, want %q", tc.title, tc.originCountry, kana, tc.expectedKana)
			}
		})
	}
}

func TestCalculateInitials_Japanese(t *testing.T) {
	testCases := []struct {
		title          string
		originCountry  string
		expectedAZ     string
		expectedKana   string
	}{
		{"さくら", "JP", "S", "さ"},
		{"カラオケ", "JP", "K", "か"},
		{"東京", "JP", "T", "た"},
		{"ありがとう", "JP", "A", "あ"},
		{"こんにちは", "JP", "K", "か"},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			az, kana := CalculateInitials(tc.title, tc.originCountry)
			if az != tc.expectedAZ {
				t.Errorf("CalculateInitials(%q, %q) az = %q, want %q", tc.title, tc.originCountry, az, tc.expectedAZ)
			}
			if kana != tc.expectedKana {
				t.Errorf("CalculateInitials(%q, %q) kana = %q, want %q", tc.title, tc.originCountry, kana, tc.expectedKana)
			}
		})
	}
}

func TestCalculateInitials_English(t *testing.T) {
	testCases := []struct {
		title          string
		originCountry  string
		expectedAZ     string
		expectedKana   string
	}{
		{"Stairway to Heaven", "US", "S", "S"},
		{"Hotel California", "US", "H", "H"},
		{"Bohemian Rhapsody", "GB", "B", "B"},
		{"Wonderwall", "GB", "W", "W"},
		{"American Pie", "CA", "A", "A"},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			az, kana := CalculateInitials(tc.title, tc.originCountry)
			if az != tc.expectedAZ {
				t.Errorf("CalculateInitials(%q, %q) az = %q, want %q", tc.title, tc.originCountry, az, tc.expectedAZ)
			}
			if kana != tc.expectedKana {
				t.Errorf("CalculateInitials(%q, %q) kana = %q, want %q", tc.title, tc.originCountry, kana, tc.expectedKana)
			}
		})
	}
}

func TestCalculateInitials_EmptyTitle(t *testing.T) {
	az, kana := CalculateInitials("", "US")
	if az != "#" || kana != "#" {
		t.Errorf("CalculateInitials('', 'US') = (%q, %q), want ('#', '#')", az, kana)
	}
}

func TestCalculateInitials_UnknownCountry(t *testing.T) {
	testCases := []struct {
		title string
		desc  string
	}{
		{"Hello World", "Latin text"},
		{"こんにちは", "Japanese text"},
		{"你好", "Chinese text"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			az, kana := CalculateInitials(tc.title, "XX")
			// Should use heuristic detection
			if az == "" || kana == "" {
				t.Errorf("CalculateInitials(%q, 'XX') returned empty strings", tc.title)
			}
		})
	}
}

func TestIsKana(t *testing.T) {
	testCases := []struct {
		char     rune
		expected bool
	}{
		{'あ', true},
		{'ア', true},
		{'か', true},
		{'カ', true},
		{'A', false},
		{'中', false},
		{'1', false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.char), func(t *testing.T) {
			result := isKana(tc.char)
			if result != tc.expected {
				t.Errorf("isKana(%q) = %v, want %v", tc.char, result, tc.expected)
			}
		})
	}
}

func TestContainsKana(t *testing.T) {
	testCases := []struct {
		text     string
		expected bool
	}{
		{"こんにちは", true},
		{"カラオケ", true},
		{"Hello World", false},
		{"你好", false},
		{"Helloあ", true},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.text, func(t *testing.T) {
			result := containsKana(tc.text)
			if result != tc.expected {
				t.Errorf("containsKana(%q) = %v, want %v", tc.text, result, tc.expected)
			}
		})
	}
}

func TestIsHanzi(t *testing.T) {
	testCases := []struct {
		char     rune
		expected bool
	}{
		{'中', true},
		{'国', true},
		{'你', true},
		{'好', true},
		{'あ', false},
		{'A', false},
		{'1', false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.char), func(t *testing.T) {
			result := isHanzi(tc.char)
			if result != tc.expected {
				t.Errorf("isHanzi(%q) = %v, want %v", tc.char, result, tc.expected)
			}
		})
	}
}

func TestIsLatin(t *testing.T) {
	testCases := []struct {
		char     rune
		expected bool
	}{
		{'A', true},
		{'Z', true},
		{'a', true},
		{'z', true},
		{'中', false},
		{'あ', false},
		{'1', false},
		{'@', false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.char), func(t *testing.T) {
			result := isLatin(tc.char)
			if result != tc.expected {
				t.Errorf("isLatin(%q) = %v, want %v", tc.char, result, tc.expected)
			}
		})
	}
}

func TestCalculateChineseInitials(t *testing.T) {
	testCases := []struct {
		title      string
		expectedAZ string
	}{
		{"青春", "Q"},
		{"爱情", "A"},
		{"北京", "B"},
		{"中国", "Z"},
		{"Hello", "H"}, // Latin fallback
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			az, kana := calculateChineseInitials(tc.title)
			if az != tc.expectedAZ {
				t.Errorf("calculateChineseInitials(%q) az = %q, want %q", tc.title, az, tc.expectedAZ)
			}
			if kana != "#" {
				t.Errorf("calculateChineseInitials(%q) kana = %q, want '#'", tc.title, kana)
			}
		})
	}
}

func TestCalculateLatinInitials(t *testing.T) {
	testCases := []struct {
		title      string
		expectedAZ string
	}{
		{"Stairway", "S"},
		{"Hotel", "H"},
		{"Bohemian", "B"},
		{"Wonderwall", "W"},
		{"American", "A"},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			az, kana := calculateLatinInitials(tc.title)
			if az != tc.expectedAZ {
				t.Errorf("calculateLatinInitials(%q) az = %q, want %q", tc.title, az, tc.expectedAZ)
			}
			if kana != tc.expectedAZ {
				t.Errorf("calculateLatinInitials(%q) kana = %q, want %q", tc.title, kana, tc.expectedAZ)
			}
		})
	}
}

func TestKanaToRomajiMapping(t *testing.T) {
	testCases := []struct {
		kana     rune
		expected string
	}{
		{'あ', "A"},
		{'か', "K"},
		{'さ', "S"},
		{'た', "T"},
		{'な', "N"},
		{'は', "H"},
		{'ま', "M"},
		{'や', "Y"},
		{'ら', "R"},
		{'わ', "W"},
		{'が', "G"},
		{'ざ', "Z"},
		{'だ', "D"},
		{'ば', "B"},
		{'ぱ', "P"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.kana), func(t *testing.T) {
			result, ok := kanaToRomaji[tc.kana]
			if !ok {
				t.Errorf("kanaToRomaji[%q] not found", tc.kana)
			}
			if result != tc.expected {
				t.Errorf("kanaToRomaji[%q] = %q, want %q", tc.kana, result, tc.expected)
			}
		})
	}
}

func TestKanaToRowMapping(t *testing.T) {
	testCases := []struct {
		kana     rune
		expected string
	}{
		{'あ', "あ"},
		{'い', "あ"},
		{'か', "か"},
		{'き', "か"},
		{'さ', "さ"},
		{'し', "さ"},
		{'た', "た"},
		{'ち', "た"},
		{'な', "な"},
		{'に', "な"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.kana), func(t *testing.T) {
			result, ok := kanaToRow[tc.kana]
			if !ok {
				t.Errorf("kanaToRow[%q] not found", tc.kana)
			}
			if result != tc.expected {
				t.Errorf("kanaToRow[%q] = %q, want %q", tc.kana, result, tc.expected)
			}
		})
	}
}

func TestCalculateInitials_MixedContent(t *testing.T) {
	testCases := []struct {
		title         string
		originCountry string
		desc          string
	}{
		{"Hello 世界", "US", "English with Chinese"},
		{"こんにちは World", "JP", "Japanese with English"},
		{"中国 Music", "CN", "Chinese with English"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			az, kana := CalculateInitials(tc.title, tc.originCountry)
			// Should not return empty strings
			if az == "" || kana == "" {
				t.Errorf("CalculateInitials(%q, %q) returned empty strings", tc.title, tc.originCountry)
			}
		})
	}
}
