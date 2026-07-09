package main

import (
	"embed"
	"mochila-archive-viewer/src/appshell"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := appshell.NewApp()

	err := wails.Run(&options.App{
		Title:            "Mochila",
		Width:            1360,
		Height:           860,
		MinWidth:         1080,
		MinHeight:        720,
		DisableResize:    false,
		AssetServer:      &assetserver.Options{Assets: assets, Handler: app},
		BackgroundColour: &options.RGBA{R: 251, G: 250, B: 242, A: 1},
		OnStartup:        app.Startup,
		Bind:             []interface{}{app},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
