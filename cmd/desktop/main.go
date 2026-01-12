package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/Bowl42/maxx-next/internal/core"
	"github.com/Bowl42/maxx-next/internal/desktop"
)

//go:embed all:../web/dist
//go:embed ../web/dist
var assets embed.FS

func main() {
	err := wails.Run(&options.App{
		Title:     "maxx-next",
		Width:      1280,
		Height:     800,
		MinWidth:   1024,
		MinHeight:  600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:   app.Startup,
		OnShutdown:  app.Shutdown,
		OnDomReady:  app.DomReady,
		OnBeforeClose: app.BeforeClose,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIconing: false,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:               false,
				HideTitleBar:            false,
				FullSizeContent:         true,
				UseToolbar:              false,
				HiddenInset:             false,
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

var app *desktop.DesktopApp

func init() {
	var err error
	app, err = desktop.NewDesktopApp()
	if err != nil {
		log.Fatalf("Failed to initialize desktop app: %v", err)
	}
}

func Startup(ctx context.Context) {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("[Desktop] Application starting up...")

	app.Startup(ctx)
}

func Shutdown(ctx context.Context) {
	log.Println("[Desktop] Application shutting down...")

	app.Shutdown(ctx)
}

func DomReady(ctx context.Context) {
	app.DomReady(ctx)
}

func BeforeClose(ctx context.Context) (prevent bool) {
	prevent = app.BeforeClose(ctx)
	return prevent
}
