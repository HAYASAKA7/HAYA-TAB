package main

import (
	"embed"
	"haya-tab/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	application := app.NewApp()

	// Start local file server
	port, err := application.StartFileServer()
	if err != nil {
		println("Error starting file server:", err.Error())
		// In a GUI app, we might want to show a dialog, but main() runs before wails.Run,
		// so we can't use wails runtime dialogs yet. Standard output is the best effort here.
		return
	}
	application.SetFileServerPort(port)

	// Create file handler for streaming
	fileHandler := app.NewFileHandler(application)

	// Create application with options
	err = wails.Run(&options.App{
		Title:            "HAYA-TAB",
		Width:            1024,
		Height:           768,
		WindowStartState: options.Maximised,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: fileHandler,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        application.Startup,
		OnDomReady:       application.DomReady,
		OnShutdown:       application.Shutdown,
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
