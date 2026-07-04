package brightness

import (
	"context"
	"log/slog"
	"sync"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/config"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

type Service struct {
	log               *slog.Logger
	bus               *eventbus.Bus
	cfg               func() config.Config
	mu                sync.Mutex
	lastInternalLevel int
	lastExternalLevel int
}

func New(log *slog.Logger, bus *eventbus.Bus, cfg func() config.Config) *Service {
	return &Service{
		log:               log,
		bus:               bus,
		cfg:               cfg,
		lastInternalLevel: 50,
		lastExternalLevel: 80,
	}
}

func (s *Service) Name() string {
	return "brightness"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("brightness service starting")
	// Populate initial levels
	if state, err := s.GetBrightness(); err == nil {
		if bs, ok := state.(BrightnessState); ok {
			s.mu.Lock()
			s.lastInternalLevel = bs.Internal
			s.lastExternalLevel = bs.External
			s.mu.Unlock()
		}
	}
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("brightness service stopping")
	return nil
}

type BrightnessState struct {
	Internal int `json:"internal"` // 0-100
	External int `json:"external"` // 0-100, if detected
}

type SetBrightnessPayload struct {
	Type  string `json:"type"`  // "internal" or "external"
	Level int    `json:"level"` // 0-100
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
		err := s.SetBrightness(payload.Type, payload.Level)
		if err != nil {
			return nil, err
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
