//go:build !ios && !android

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS is a no-op on non-iOS non-Android platforms.
func modifyOptionsForIOS(_ *application.Options) {}
