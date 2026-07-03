package power

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
	return "power"
}

func (s *Service) Start(ctx context.Context) error {
	s.log.Info("power service starting")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.log.Info("power service stopping")
	return nil
}

func (s *Service) Handle(ctx context.Context, req protocol.Envelope) (any, error) {
	switch req.Action {
	case "lock":
		return nil, s.Lock()
	case "sleep":
		return nil, s.Sleep()
	case "restart":
		return nil, s.Restart()
	case "shutdown":
		return nil, s.Shutdown()
	default:
		return nil, &protocol.Error{Code: protocol.CodeUnsupported, Message: "unknown power action"}
	}
}
