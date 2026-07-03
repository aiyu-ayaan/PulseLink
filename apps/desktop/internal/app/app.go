// Package app is the composition root: it wires configuration, logging, the
// event bus, storage, networking and feature services together.
//
// This is the only place that knows about all the pieces. Everything else
// depends on narrow interfaces, so services stay testable in isolation.
package app

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/logging"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/security"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/service"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/storage"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/transport"
)

// Version is the backend build version, advertised in the handshake.
const Version = "0.1.0-dev"

// Topic names published on the event bus.
const (
	// TopicPresence carries the []string of connected device IDs on change.
	TopicPresence = "presence.changed"
)

// App holds the wired-up backend and its lifecycle.
type App struct {
	cfgPath  string
	cfg      config.Config
	log      *slog.Logger
	bus      *eventbus.Bus
	store    *storage.Store
	registry *service.Registry
	router   *transport.Router
	hub      *transport.Hub
	server   *transport.Server
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

	router := transport.NewRouter()
	hub := transport.NewHub()

	a := &App{
		cfgPath:  cfgPath,
		cfg:      cfg,
		log:      log,
		bus:      bus,
		store:    store,
		registry: reg,
		router:   router,
		hub:      hub,
	}

	// Publish presence changes so the future dashboard can react.
	hub.OnChange = func(deviceIDs []string) {
		bus.Publish(eventbus.Event{Topic: TopicPresence, Payload: deviceIDs})
	}

	if err := a.buildServer(); err != nil {
		store.Close()
		return nil, err
	}
	a.registerServices()
	return a, nil
}

// buildServer constructs the WebSocket server, including TLS when enabled.
func (a *App) buildServer() error {
	addr := net.JoinHostPort(a.cfg.Server.Host, strconv.Itoa(a.cfg.Server.Port))

	scfg := transport.Config{
		Addr: addr,
		Info: transport.ServerInfo{Name: a.cfg.DeviceName, Version: Version},
	}
	if a.cfg.Server.EnableTLS {
		tc, err := security.SelfSignedTLS([]string{a.cfg.DeviceName})
		if err != nil {
			return err
		}
		scfg.TLS = tc
	}

	// AllowAll is a placeholder; the auth package replaces it in the next chunk.
	a.server = transport.NewServer(scfg, a.log, a.hub, a.router, transport.AllowAll{})
	return nil
}

// registerServices constructs and registers every feature module. New modules
// are added here as the backend grows.
func (a *App) registerServices() {
	// Services are registered in later chunks (media, brightness, power, ...).
}

// Start brings up storage-backed services and the network server.
func (a *App) Start(ctx context.Context) error {
	a.log.Info("PulseLink backend starting",
		"device", a.cfg.DeviceName,
		"version", Version,
		"protocol", protocol.Version,
		"services", a.registry.Names(),
	)
	if err := a.registry.StartAll(ctx); err != nil {
		return err
	}
	return a.server.Start(ctx)
}

// Stop shuts the server and services down. It uses a background context so
// shutdown proceeds even after the run context is cancelled.
func (a *App) Stop() {
	a.log.Info("PulseLink backend stopping")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.server.Stop(shutdownCtx); err != nil {
		a.log.Error("stopping server", "err", err)
	}
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

// Hub exposes the connection hub for the UI/API layer.
func (a *App) Hub() *transport.Hub { return a.hub }
