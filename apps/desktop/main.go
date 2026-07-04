//go:build wails

// Command pulselink is the Wails v3 desktop application: a native Windows
// window that hosts the React UI and runs the PulseLink backend in-process.
//
// It is guarded by the `wails` build tag so the rest of the module builds and
// tests without the Wails dependency. Build it with:
//
//	go build -tags wails -o pulselink.exe ./apps/desktop
//
// See docs/desktop-app.md for the full toolchain setup.
package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/app"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
)

// The compiled frontend is embedded so the app is a single self-contained
// binary. Run `npm run build` in apps/desktop/frontend first.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// The embedded UI is served over http:// inside WebView2, so the frontend
	// connects with ws://. Force the loopback backend to plain ws for the MVP;
	// it is loopback-only here. TLS (the self-signed cert code already exists)
	// is a follow-up once cert trust is wired for LAN/Android.
	cfgPath := filepath.Join(config.DataDir(), "config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.Server.EnableTLS {
		cfg.Server.EnableTLS = false
		if err := config.Save(cfgPath, cfg); err != nil {
			log.Fatalf("config save: %v", err)
		}
	}

	backend, err := app.New(cfgPath)
	if err != nil {
		log.Fatalf("backend init: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := backend.Start(ctx); err != nil {
		log.Fatalf("backend start: %v", err)
	}

	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatalf("assets: %v", err)
	}

	wapp := application.New(application.Options{
		Name:        "PulseLink",
		Description: "Control your Windows PC from your phone",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(dist),
		},
	})

	wapp.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:            "PulseLink",
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        640,
		BackgroundColour: application.NewRGB(27, 27, 31),
		URL:              "/",
	})

	// Tear the backend down when the window/app closes.
	defer backend.Stop()
	if err := wapp.Run(); err != nil {
		log.Fatalf("wails run: %v", err)
	}
}
