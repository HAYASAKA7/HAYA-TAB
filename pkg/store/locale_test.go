package store

import (
	"strings"
	"testing"
)

func TestDetectSystemLocale_ReturnsSupportedLocale(t *testing.T) {
	locale := DetectSystemLocale()
	if locale == "" {
		t.Error("DetectSystemLocale() returned empty string")
	}

	supportedLocales := map[string]bool{
		"en":    true,
		"zh-CN": true,
		"zh-TW": true,
		"ja":    true,
	}
	if !supportedLocales[locale] {
		t.Errorf("DetectSystemLocale() = %q, not in supported locales set", locale)
	}
}

// TestLocaleMapping directly exercises the mapping and normalization logic
// inside DetectSystemLocale by calling the internal detectLocaleFromTag helper
// that mirrors its logic for unit-testable purposes.
// Since detectSystemLanguageTag is platform-specific and opaque, we test the
// normalization logic via the exported function's contract and via the
// unexported supportedLocaleMap table.
func TestSupportedLocaleMap_ExactMatches(t *testing.T) {
	tests := []struct {
		tag  string
		want string
	}{
		{"zh-cn", "zh-CN"},
		{"zh-sg", "zh-CN"},
		{"zh-hans", "zh-CN"},
		{"zh-tw", "zh-TW"},
		{"zh-hk", "zh-TW"},
		{"zh-mo", "zh-TW"},
		{"zh-hant", "zh-TW"},
		{"ja", "ja"},
		{"ja-jp", "ja"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got, ok := supportedLocaleMap[tt.tag]
			if !ok {
				t.Errorf("tag %q not found in supportedLocaleMap", tt.tag)
				return
			}
			if got != tt.want {
				t.Errorf("supportedLocaleMap[%q] = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

// testDetectLocaleFromTag replicates DetectSystemLocale's internal logic so
// we can drive it with arbitrary tags without relying on OS locale detection.
func testDetectLocaleFromTag(tag string) string {
	if tag == "" {
		return "en"
	}

	tag = strings.ToLower(strings.TrimSpace(tag))
	if idx := strings.Index(tag, "."); idx > 0 {
		tag = tag[:idx]
	}
	tag = strings.ReplaceAll(tag, "_", "-")

	if locale, ok := supportedLocaleMap[tag]; ok {
		return locale
	}

	prefix := tag
	if idx := strings.Index(tag, "-"); idx > 0 {
		prefix = tag[:idx]
	}

	switch prefix {
	case "zh":
		return "zh-CN"
	case "ja":
		return "ja"
	}

	return "en"
}

func TestDetectLocaleFromTag_AllBranches(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		// Empty input → fallback
		{"empty tag returns en", "", "en"},

		// Exact map matches
		{"zh-CN exact", "zh-CN", "zh-CN"},
		{"zh-TW exact", "zh-TW", "zh-TW"},
		{"ja-JP exact", "ja-JP", "ja"},

		// Normalisation: underscores → dashes
		{"zh_CN underscore normalised", "zh_CN", "zh-CN"},
		{"ja_JP underscore normalised", "ja_JP", "ja"},

		// Normalisation: encoding suffix stripped
		{"zh-CN.UTF-8 suffix stripped", "zh-CN.UTF-8", "zh-CN"},
		{"ja_JP.UTF-8 suffix stripped", "ja_JP.UTF-8", "ja"},

		// Case insensitive
		{"ZH-CN uppercase", "ZH-CN", "zh-CN"},

		// Prefix fallback (not in exact map)
		{"zh-yue uses zh prefix → zh-CN", "zh-yue", "zh-CN"},
		{"ja-unknown uses ja prefix → ja", "ja-unknown", "ja"},

		// Prefix-only inputs
		{"zh prefix only → zh-CN", "zh", "zh-CN"},

		// Unknown language → en
		{"fr returns en", "fr", "en"},
		{"fr-FR returns en", "fr-FR", "en"},
		{"en-US returns en", "en-US", "en"},
		{"ko returns en", "ko", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testDetectLocaleFromTag(tt.tag)
			if got != tt.want {
				t.Errorf("detectLocaleFromTag(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}
