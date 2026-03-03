package store

import (
	"strings"
	"testing"
)

// Test the locale normalization and mapping logic directly
func TestLocaleNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Exact matches
		{"Chinese Simplified exact", "zh-cn", "zh-CN"},
		{"Chinese Traditional exact", "zh-tw", "zh-TW"},
		{"Japanese exact", "ja", "ja"},
		{"Japanese with region", "ja-jp", "ja"},

		// Normalization tests
		{"Uppercase input", "ZH-CN", "zh-CN"},
		{"Mixed case", "Zh-Cn", "zh-CN"},
		{"With encoding suffix", "zh-cn.utf8", "zh-CN"},
		{"With underscore", "zh_cn", "zh-CN"},
		{"Underscore and encoding", "zh_CN.UTF-8", "zh-CN"},
		{"With whitespace", " zh-cn ", "zh-CN"},

		// Special locale codes
		{"Singapore Chinese", "zh-sg", "zh-CN"},
		{"Hong Kong Chinese", "zh-hk", "zh-TW"},
		{"Macau Chinese", "zh-mo", "zh-TW"},
		{"Simplified script", "zh-hans", "zh-CN"},
		{"Traditional script", "zh-hant", "zh-TW"},

		// Prefix matches
		{"Chinese prefix", "zh", "zh-CN"},
		{"Chinese unknown region", "zh-xx", "zh-CN"},
		{"Japanese unknown region", "ja-xx", "ja"},

		// Unsupported locales
		{"English", "en", "en"},
		{"English US", "en-us", "en"},
		{"French", "fr", "en"},
		{"German", "de", "en"},
		{"Spanish", "es", "en"},
		{"Korean", "ko", "en"},
		{"Unknown locale", "xx-yy", "en"},
		{"Empty string", "", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the normalization logic from DetectSystemLocale
			tag := tt.input

			// Normalize
			tag = strings.ToLower(strings.TrimSpace(tag))
			if idx := strings.Index(tag, "."); idx > 0 {
				tag = tag[:idx]
			}
			tag = strings.ReplaceAll(tag, "_", "-")

			var result string

			// Exact match
			if locale, ok := supportedLocaleMap[tag]; ok {
				result = locale
			} else {
				// Prefix match
				prefix := tag
				if idx := strings.Index(tag, "-"); idx > 0 {
					prefix = tag[:idx]
				}

				switch prefix {
				case "zh":
					result = "zh-CN"
				case "ja":
					result = "ja"
				default:
					result = "en"
				}
			}

			if result != tt.expected {
				t.Errorf("Normalized locale for %q = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectSystemLocale_Integration(t *testing.T) {
	// Test with actual system locale detection
	result := DetectSystemLocale()

	// Result should be one of the supported locales or "en"
	validLocales := map[string]bool{
		"zh-CN": true,
		"zh-TW": true,
		"ja":    true,
		"en":    true,
	}

	if !validLocales[result] {
		t.Errorf("DetectSystemLocale() returned unexpected locale: %v", result)
	}
}

func TestSupportedLocaleMap(t *testing.T) {
	// Verify the supported locale map has expected entries
	expectedMappings := map[string]string{
		"zh-cn":   "zh-CN",
		"zh-tw":   "zh-TW",
		"zh-hk":   "zh-TW",
		"zh-hans": "zh-CN",
		"zh-hant": "zh-TW",
		"ja":      "ja",
		"ja-jp":   "ja",
	}

	for key, expectedValue := range expectedMappings {
		if actualValue, ok := supportedLocaleMap[key]; !ok {
			t.Errorf("supportedLocaleMap missing key: %s", key)
		} else if actualValue != expectedValue {
			t.Errorf("supportedLocaleMap[%s] = %s, want %s", key, actualValue, expectedValue)
		}
	}
}
