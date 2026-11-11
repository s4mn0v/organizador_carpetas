package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Organizador de carpetas",
		Width:     1360,
		Height:    860,
		MinWidth:  1024,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 238, G: 241, B: 243, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown, // apaga el servidor MCP de Claude
		// Arrastrar carpetas desde el Explorador al diálogo "Subir carpeta" (solo en zonas marcadas
		// con --wails-drop-target; el arrastre interno del árbol sigue igual).
		DragAndDrop: &options.DragAndDrop{EnableFileDrop: true},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
