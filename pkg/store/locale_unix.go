//go:build !windows

package store

import "os"

// detectSystemLanguageTag returns the locale tag from environment variables on Unix-like systems.
func detectSystemLanguageTag() string {
	for _, env := range []string{"LANG", "LC_ALL", "LANGUAGE"} {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return ""
}
