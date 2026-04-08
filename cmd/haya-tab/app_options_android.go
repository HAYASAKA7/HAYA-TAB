//go:build android

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS adjusts the application options for Android.
// Reusing the same function name for consistency with the platform-agnostic build.
func modifyOptionsForIOS(opts *application.Options) {
	// Disable signal handlers on Android to prevent crashes.
	opts.DisableDefaultSignalHandler = true
}
