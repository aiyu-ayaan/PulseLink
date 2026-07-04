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

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/auth"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/logging"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/security"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/service"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/storage"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/transport"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/apps"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/brightness"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/clipboard"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/devices"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/filetransfer"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/input"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/media"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/notification"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/pairing"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/power"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/settings"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/sysinfo"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/services/volume"
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
		info := hub.ConnectedDevicesInfo()
		bus.Publish(eventbus.Event{Topic: TopicPresence, Payload: info})
	}

	// Forward presence/devices changes from the eventbus to connected devices.
	bus.Subscribe(TopicPresence, func(ev eventbus.Event) {
		payload, ok := ev.Payload.([]transport.DeviceInfo)
		if !ok {
			return
		}
		env, err := protocol.NewEvent("devices", "changed", payload)
		if err == nil {
			hub.Broadcast(env)
		}
	})

	// Forward clipboard changes from the eventbus to connected devices.
	bus.Subscribe("clipboard.changed", func(ev eventbus.Event) {
		payload, ok := ev.Payload.(clipboard.ClipboardPayload)
		if !ok {
			return
		}
		env, err := protocol.NewEvent("clipboard", "changed", payload)
		if err == nil {
			hub.Broadcast(env)
		}
	})

	// Forward volume changes from the eventbus to connected devices.
	bus.Subscribe("volume.changed", func(ev eventbus.Event) {
		payload, ok := ev.Payload.(volume.VolumeState)
		if !ok {
			return
		}
		env, err := protocol.NewEvent("volume", "changed", payload)
		if err == nil {
			hub.Broadcast(env)
		}
	})

	// Forward pairing requests from the eventbus to connected devices.
	bus.Subscribe("pairing.request", func(ev eventbus.Event) {
		payload, ok := ev.Payload.(transport.DeviceInfo)
		if !ok {
			log.Warn("pairing.request dropped: unexpected payload type")
			return
		}
		log.Info("pairing.request event", "device", payload.ID, "name", payload.Name)
		env, err := protocol.NewEvent("pairing", "request", payload)
		if err != nil {
			log.Error("pairing.request encode", "err", err)
			return
		}
		hub.Broadcast(env)
	})

	// Forward pairing approvals to the specific device.
	bus.Subscribe("pairing.approved", func(ev eventbus.Event) {
		deviceID, ok := ev.Payload.(string)
		if !ok {
			log.Warn("pairing.approved dropped: unexpected payload type")
			return
		}
		log.Info("pairing.approved event", "device", deviceID)
		env, err := protocol.NewEvent("pairing", "approved", nil)
		if err != nil {
			log.Error("pairing.approved encode", "err", err)
			return
		}
		hub.SendToDevice(deviceID, env)
	})

	// Handle pairing rejections by disconnecting the device.
	bus.Subscribe("pairing.rejected", func(ev eventbus.Event) {
		deviceID, ok := ev.Payload.(string)
		if !ok {
			log.Warn("pairing.rejected dropped: unexpected payload type")
			return
		}
		log.Info("pairing.rejected event", "device", deviceID)
		hub.DisconnectDevice(deviceID)
	})

	// Log presence changes for debugging.
	bus.Subscribe(TopicPresence, func(ev eventbus.Event) {
		payload, ok := ev.Payload.([]transport.DeviceInfo)
		if !ok {
			return
		}
		ids := make([]string, len(payload))
		for i, d := range payload {
			ids[i] = d.ID
		}
		log.Debug("presence changed", "connected", ids)
	})

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

	// Use the real database-backed Authenticator
	authSvc := auth.New(a.store, a.bus)
	a.server = transport.NewServer(scfg, a.log, a.hub, a.router, authSvc)
	return nil
}

// registerServices constructs and registers every feature module. New modules
// are added here as the backend grows.
func (a *App) registerServices() {
	mediaSvc := media.New(a.log, a.bus)
	a.registry.Register(mediaSvc)
	a.router.Register(mediaSvc.Name(), mediaSvc)

	volumeSvc := volume.New(a.log, a.bus)
	a.registry.Register(volumeSvc)
	a.router.Register(volumeSvc.Name(), volumeSvc)

	devicesSvc := devices.New(a.log, a.bus, a.hub, a.store)
	a.registry.Register(devicesSvc)
	a.router.Register(devicesSvc.Name(), devicesSvc)

	pairingSvc := pairing.New(a.log, a.bus, a.store, a.hub)
	a.registry.Register(pairingSvc)
	a.router.Register(pairingSvc.Name(), pairingSvc)

	brightnessSvc := brightness.New(a.log, a.bus)
	a.registry.Register(brightnessSvc)
	a.router.Register(brightnessSvc.Name(), brightnessSvc)

	clipboardSvc := clipboard.New(a.log, a.bus)
	a.registry.Register(clipboardSvc)
	a.router.Register(clipboardSvc.Name(), clipboardSvc)

	powerSvc := power.New(a.log, a.bus)
	a.registry.Register(powerSvc)
	a.router.Register(powerSvc.Name(), powerSvc)

	sysinfoSvc := sysinfo.New(a.log, a.bus)
	a.registry.Register(sysinfoSvc)
	a.router.Register(sysinfoSvc.Name(), sysinfoSvc)

	appsSvc := apps.New(a.log, a.bus)
	a.registry.Register(appsSvc)
	a.router.Register(appsSvc.Name(), appsSvc)

	inputSvc := input.New(a.log, a.bus)
	a.registry.Register(inputSvc)
	a.router.Register(inputSvc.Name(), inputSvc)

	notificationSvc := notification.New(a.log, a.bus)
	a.registry.Register(notificationSvc)
	a.router.Register(notificationSvc.Name(), notificationSvc)

	filetransferSvc := filetransfer.New(a.log, a.bus)
	a.registry.Register(filetransferSvc)
	a.router.Register(filetransferSvc.Name(), filetransferSvc)

	settingsSvc := settings.New(a.log, a.bus, a.cfgPath)
	a.registry.Register(settingsSvc)
	a.router.Register(settingsSvc.Name(), settingsSvc)
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
