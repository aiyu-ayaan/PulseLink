// Package app is the composition root: it wires configuration, logging, the
// event bus, storage, networking and feature services together.
//
// This is the only place that knows about all the pieces. Everything else
// depends on narrow interfaces, so services stay testable in isolation.
package app

import (
	"context"
	"log/slog"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/logging"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/service"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/storage"
)

// App holds the wired-up backend and its lifecycle.
type App struct {
	cfgPath  string
	cfg      config.Config
	log      *slog.Logger
	bus      *eventbus.Bus
	store    *storage.Store
	registry *service.Registry
}

// New loads configuration from cfgPath and constructs the application graph
// without starting anything.
func New(cfgPath string) (*App, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	log := logging.New(cfg.LogLevel)
	bus := eventbus.New()
	reg := service.NewRegistry(log)

	store, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	a := &App{
		cfgPath:  cfgPath,
		cfg:      cfg,
		log:      log,
		bus:      bus,
		store:    store,
		registry: reg,
	}
	a.registerServices()
	return a, nil
}

// registerServices constructs and registers every feature module. New modules
// are added here as the backend grows.
func (a *App) registerServices() {
	// Services are registered in later chunks (media, brightness, power, ...).
}

// Start brings up all services. The passed context governs their lifetime.
func (a *App) Start(ctx context.Context) error {
	a.log.Info("PulseLink backend starting",
		"device", a.cfg.DeviceName,
		"services", a.registry.Names(),
	)
	return a.registry.StartAll(ctx)
}

// Stop shuts services down. It uses a background context so shutdown proceeds
// even after the run context is cancelled.
func (a *App) Stop() {
	a.log.Info("PulseLink backend stopping")
	_ = a.registry.StopAll(context.Background())
	if err := a.store.Close(); err != nil {
		a.log.Error("closing store", "err", err)
	}
}

// Config exposes the loaded configuration (read-only use by the future UI).
func (a *App) Config() config.Config { return a.cfg }

// Bus exposes the event bus for the UI/API layer.
func (a *App) Bus() *eventbus.Bus { return a.bus }

// Store exposes the data layer for the UI/API layer.
func (a *App) Store() *storage.Store { return a.store }
