package media

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
	return "media"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("media service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("media service stopping")
	return nil
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "play_pause":
		return nil, s.PlayPause()
	case "next":
		return nil, s.Next()
	case "previous":
		return nil, s.Previous()
	case "stop":
		return nil, s.StopMedia()
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown media action"}
	}
}
