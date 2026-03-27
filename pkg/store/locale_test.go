package store

import "testing"

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
