package brightness

import (
	"context"
	"log/slog"

	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/eventbus"
	"github.com/aiyu-ayaan/pulselink/apps/desktop/internal/protocol"
)

type Service struct {
	log *slog.Logger
	bus *eventbus.Bus
}

func New(log *slog.Logger, bus *eventbus.Bus) *Service {
	return &Service{
		log: log,
		bus: bus,
	}
}

func (s *Service) Name() string {
	return "brightness"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("brightness service starting")
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
		return nil, s.SetBrightness(payload.Type, payload.Level)
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown brightness action"}
	}
}
