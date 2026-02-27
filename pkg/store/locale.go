package store

import "strings"

// supportedLocaleMap maps system locale tags (lowercase) to supported app locale codes.
var supportedLocaleMap = map[string]string{
	"zh-cn":   "zh-CN",
	"zh-sg":   "zh-CN",
	"zh-hans": "zh-CN",
	"zh-tw":   "zh-TW",
	"zh-hk":   "zh-TW",
	"zh-mo":   "zh-TW",
	"zh-hant": "zh-TW",
	"ja":      "ja",
	"ja-jp":   "ja",
}

// DetectSystemLocale detects the OS language and maps it to a supported app locale.
// Returns "en" if the system language is not in the supported set.
func DetectSystemLocale() string {
	tag := detectSystemLanguageTag()
	if tag == "" {
		return "en"
	}

	// Normalize: lowercase, strip encoding suffix, unify separator
	tag = strings.ToLower(strings.TrimSpace(tag))
	if idx := strings.Index(tag, "."); idx > 0 {
		tag = tag[:idx]
	}
	tag = strings.ReplaceAll(tag, "_", "-")

	// Exact match
	if locale, ok := supportedLocaleMap[tag]; ok {
		return locale
	}

	// Prefix match (e.g. "zh" → "zh-CN", "ja-xx" → "ja")
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
