//go:build ios

package main

import (
	"haya-tab/pkg/store"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// modifyOptionsForIOS adjusts the application options for iOS.
func modifyOptionsForIOS(opts *application.Options) {
	applyIOSOptions(opts, store.DetectSystemLocale())
}
