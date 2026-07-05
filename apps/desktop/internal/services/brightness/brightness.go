package brightness

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

// MonitorBrightness describes a single monitor's brightness.
type MonitorBrightness struct {
	ID     string `json:"id"`     // e.g. "monitor-0"
	Name   string `json:"name"`   // e.g. "Display 1 (Primary)"
	Level  int    `json:"level"`  // 0-100
	Method string `json:"method"` // "wmi", "ddc", or "gamma"
}

// BrightnessState is the response payload for brightness.get.
type BrightnessState struct {
	Monitors []MonitorBrightness `json:"monitors"`
}

// SetBrightnessPayload is the request payload for brightness.set.
type SetBrightnessPayload struct {
	Monitor string `json:"monitor"` // monitor id, "all", or legacy "internal"/"external"
	Level   int    `json:"level"`   // 0-100
	Type    string `json:"type"`    // legacy alias for Monitor
}

type Service struct {
	log          *slog.Logger
	bus          *eventbus.Bus
	cfg          func() config.Config
	mu           sync.Mutex
	probeMu      sync.Mutex // serialises probes (they do slow hardware reads outside mu)
	lastMonitors []MonitorBrightness
	lastLevels   map[string]int       // id -> last known level
	methodCache  map[string]string    // id -> "wmi"/"ddc"/"gamma"
	ddcRange     map[string][2]uint32 // id -> {min,max} raw VCP range reported by the monitor
	pending      map[string]int       // id -> newest requested level awaiting hardware write
	inFlight     map[string]bool      // id -> a worker is draining pending for this monitor
	gammaDim     map[string]bool      // id -> gamma ramp currently dimmed by a fallback
	monitorHW    []monitorHW          // cached hardware handles (Windows only, empty on other OS)
	probed       bool
	refreshing   bool
	lastRefresh  time.Time
}

func New(log *slog.Logger, bus *eventbus.Bus, cfg func() config.Config) *Service {
	return &Service{
		log:         log,
		bus:         bus,
		cfg:         cfg,
		lastLevels:  make(map[string]int),
		methodCache: make(map[string]string),
		ddcRange:    make(map[string][2]uint32),
		pending:     make(map[string]int),
		inFlight:    make(map[string]bool),
		gammaDim:    make(map[string]bool),
	}
}

func (s *Service) Name() string {
	return "brightness"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("brightness service starting")
	// Probe monitors once (populates cache). Non-fatal.
	s.GetBrightness()

	s.bus.Subscribe("settings.changed", func(evt eventbus.Event) {
		s.mu.Lock()
		s.probed = false
		s.mu.Unlock()
	})
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("brightness service stopping")
	return nil
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "get":
		return s.GetBrightness()
	case "set":
		var payload SetBrightnessPayload
		if err := req.Bind(&payload); err != nil {
			return nil, &protocol.Error{Code: protocol.CodeBadRequest, Message: "malformed set brightness payload"}
		}
		monitorID := payload.Monitor
		if monitorID == "" {
			monitorID = payload.Type
		}
		if monitorID == "" {
			monitorID = "all"
		}
		if err := s.SetBrightness(monitorID, payload.Level); err != nil {
			s.log.Error("set brightness failed", "err", err)
		}
		// Return the cached state immediately — no re-probing
		s.mu.Lock()
		state := BrightnessState{Monitors: s.lastMonitors}
		s.mu.Unlock()
		s.bus.Publish(eventbus.Event{Topic: "brightness.changed", Payload: state})
		return state, nil
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown brightness action"}
	}
}
