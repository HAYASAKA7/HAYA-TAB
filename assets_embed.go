package appassets

import "embed"

// FS embeds the built frontend assets for desktop/mobile runtime loading.
//
//go:embed all:frontend/dist
var FS embed.FS
