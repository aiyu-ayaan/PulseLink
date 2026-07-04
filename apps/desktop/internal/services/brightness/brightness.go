package brightness

import (
	"context"
	"log/slog"
	"sync"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

// MonitorBrightness describes a single monitor's brightness.
type MonitorBrightness struct {
	ID     string `json:"id"`     // e.g. "monitor-0"
	Name   string `json:"name"`   // e.g. "Display 1 (Primary)"
	Level  int    `json:"level"`  // 0-100
	Method string `json:"method"` // "ddc" or "gamma"
}

// BrightnessState is the response payload for brightness.get.
type BrightnessState struct {
	Monitors []MonitorBrightness `json:"monitors"`
}

// SetBrightnessPayload is the request payload for brightness.set.
type SetBrightnessPayload struct {
	Monitor string `json:"monitor"` // monitor id, e.g. "monitor-0", "all", or legacy "internal"/"external"
	Level   int    `json:"level"`   // 0-100
	// Legacy fields — accepted for backward compat
	Type string `json:"type"` // alias for Monitor
}

type Service struct {
	log          *slog.Logger
	bus          *eventbus.Bus
	cfg          func() config.Config
	mu           sync.Mutex
	lastMonitors []MonitorBrightness
	lastLevels   map[string]int // id -> last known level
}

func New(log *slog.Logger, bus *eventbus.Bus, cfg func() config.Config) *Service {
	return &Service{
		log:        log,
		bus:        bus,
		cfg:        cfg,
		lastLevels: make(map[string]int),
	}
}

func (s *Service) Name() string {
	return "brightness"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("brightness service starting")
	// Populate initial levels — errors are non-fatal
	if state, err := s.GetBrightness(); err == nil {
		if bs, ok := state.(BrightnessState); ok {
			s.mu.Lock()
			s.lastMonitors = bs.Monitors
			for _, m := range bs.Monitors {
				s.lastLevels[m.ID] = m.Level
			}
			s.mu.Unlock()
		}
	}
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("brightness service stopping")
	return nil
}

func (s *Service) getLastLevel(id string) int {
	if v, ok := s.lastLevels[id]; ok {
		return v
	}
	return 100 // default to full brightness
}

func (s *Service) setLastLevel(id string, level int) {
	s.lastLevels[id] = level
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
		// Support legacy "type" field as alias for "monitor"
		monitorID := payload.Monitor
		if monitorID == "" {
			monitorID = payload.Type
		}
		if monitorID == "" {
			monitorID = "all"
		}
		err := s.SetBrightness(monitorID, payload.Level)
		if err != nil {
			s.log.Error("set brightness failed", "err", err)
			// Return the current state anyway — don't crash the connection
			state, _ := s.GetBrightness()
			return state, nil
		}
		state, err := s.GetBrightness()
		if err == nil {
			s.bus.Publish(eventbus.Event{Topic: "brightness.changed", Payload: state})
		}
		return state, nil
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown brightness action"}
	}
}
