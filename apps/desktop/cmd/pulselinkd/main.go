// Command pulselinkd runs the PulseLink backend headlessly (no UI).
//
// The Wails desktop app (Stage 2) will embed the same app.App; this binary lets
// the backend run and be tested on its own.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/app"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
)

func main() {
	defaultCfg := filepath.Join(config.DataDir(), "config.json")
	cfgPath := flag.String("config", defaultCfg, "path to config.json")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.New(*cfgPath)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	if err := a.Start(ctx); err != nil {
		log.Fatalf("start: %v", err)
	}

	<-ctx.Done() // block until Ctrl+C / termination
	a.Stop()
}
