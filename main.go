package main

import (
	"embed"
	"haya-tab/internal/app"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	myApp := app.NewApp()

	// Start local file server
	port, err := myApp.StartFileServer()
	if err != nil {
		log.Fatal("Error starting file server:", err)
	}
	myApp.SetFileServerPort(port)

	// Create application with options
	appInstance := application.New(application.Options{
		Name:        "HAYA-TAB",
		Description: "A lightweight music tab manager for guitarists and musicians",
		Services: []application.Service{
			application.NewService(myApp),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Set the global app instance inside our app structure (if we rewrite it to remove context)
	myApp.SetApp(appInstance)

	// In Wails v3, we use application events for lifecycle hooks
	appInstance.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(e *application.ApplicationEvent) {
		myApp.Startup()
		myApp.DomReady() // Called after startup since WindowDomReady was removed
	})
	
	// Ensure we shut down gracefully
	appInstance.OnShutdown(func() {
		myApp.Shutdown()
	})

	// Create the main window
	window := appInstance.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "HAYA-TAB",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// Set window state (Maximised)
	window.Maximise()

	window.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		appInstance.Quit()
	})

	err = appInstance.Run()
	if err != nil {
		log.Fatal(err)
	}
}
